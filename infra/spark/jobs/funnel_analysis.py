# infra/spark/jobs/funnel_analysis.py
import os
import sys
from pyspark.sql import SparkSession
from pyspark.sql.functions import col, from_json, count, window
from pyspark.sql.types import StructType, StringType, MapType

def get_db_properties():
    db_user = os.environ.get("DB_USER")
    db_pass = os.environ.get("DB_PASSWORD")
    db_url = os.environ.get("DB_URL", "jdbc:postgresql://postgres-primary.data-postgres.svc.cluster.local:5432/ecommerce_db")
    
    if not db_user or not db_pass:
        print("Error: DB credentials not found in environment variables.")
        sys.exit(1)
        
    return db_url, {
        "user": db_user,
        "password": db_pass,
        "driver": "org.postgresql.Driver"
    }

def process_batch(df, epoch_id):
    db_url, db_props = get_db_properties()
    
    # 1. Funnel Calculation (Micro-batch aggregation)
    funnel_stats = df.groupBy("offer_id", "event_type") \
        .agg(count("event_id").alias("event_count")) \
        .orderBy("offer_id") \
        .withColumnRenamed("offer_id", "campaign_id")
    
    if funnel_stats.count() == 0:
        return
        
    # 2. Write to Staging Table
    staging_table = f"analytics.funnel_daily_stage"
    funnel_stats.write.jdbc(
        url=db_url,
        table=staging_table,
        mode="overwrite",
        properties=db_props
    )
    
    # 3. Perform UPSERT via Spark's internal JDBC connection (Py4J)
    spark = SparkSession.builder.getOrCreate()
    gw = spark.sparkContext._gateway
    java_import = gw.jvm.java.sql.DriverManager
    
    conn = None
    stmt = None
    try:
        conn = java_import.getConnection(db_url, db_props["user"], db_props["password"])
        stmt = conn.createStatement()
        
        # Ensure target table exists
        create_target = """
        CREATE SCHEMA IF NOT EXISTS analytics;
        CREATE TABLE IF NOT EXISTS analytics.funnel_daily (
            campaign_id VARCHAR(255),
            event_type VARCHAR(255),
            event_count BIGINT,
            PRIMARY KEY (campaign_id, event_type)
        );
        """
        stmt.execute(create_target)
        
        # Execute UPSERT (PostgreSQL specific)
        upsert_query = f"""
        INSERT INTO analytics.funnel_daily (campaign_id, event_type, event_count)
        SELECT campaign_id, event_type, event_count FROM {staging_table}
        ON CONFLICT (campaign_id, event_type) 
        DO UPDATE SET event_count = analytics.funnel_daily.event_count + EXCLUDED.event_count;
        """
        stmt.execute(upsert_query)
        
        # Cleanup staging table
        stmt.execute(f"DROP TABLE {staging_table}")
        
    except Exception as e:
        print(f"Error executing upsert in epoch {epoch_id}: {e}")
        raise e
    finally:
        if stmt is not None: stmt.close()
        if conn is not None: conn.close()
        
def main():
    spark = SparkSession.builder \
        .appName("FunnelAnalysisStreaming") \
        .getOrCreate()
        
    kafka_brokers = os.environ.get("KAFKA_BROKERS", "kafka-cluster-kafka-bootstrap.kafka:9092")
    topics = "page.viewed,conversion.completed,checkout.started"
    
    # 1. Read Structured Stream
    raw_stream = spark.readStream \
        .format("kafka") \
        .option("kafka.bootstrap.servers", kafka_brokers) \
        .option("subscribe", topics) \
        .option("startingOffsets", "earliest") \
        .option("failOnDataLoss", "false") \
        .load()
        
    schema = StructType() \
        .add("event_id", StringType()) \
        .add("timestamp", StringType()) \
        .add("service", StringType()) \
        .add("event_type", StringType()) \
        .add("session_id", StringType()) \
        .add("offer_id", StringType()) \
        .add("metadata", MapType(StringType(), StringType()))
        
    events_stream = raw_stream.select(
        col("topic").alias("event_type"),
        from_json(col("value").cast("string"), schema).alias("data")
    ).select(
        "event_type",
        col("data.event_id"), 
        col("data.offer_id"),
        col("data.session_id")
    ).filter(col("event_id").isNotNull())

    # 2. Write Stream with Checkpointing and ForeachBatch (Exactly-Once)
    checkpoint_loc = os.environ.get("CHECKPOINT_PATH", "s3a://ecommerce-spark-checkpoints/funnel_analysis")
    
    query = events_stream.writeStream \
        .outputMode("append") \
        .foreachBatch(process_batch) \
        .option("checkpointLocation", checkpoint_loc) \
        .trigger(processingTime="1 minute") \
        .start()
        
    query.awaitTermination()

if __name__ == "__main__":
    main()