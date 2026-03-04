module github.com/Moiz-Ali-Moomin/microservice-ecommerce-aws-cloud-eks/services/product-service

go 1.24.0

replace github.com/Moiz-Ali-Moomin/microservice-ecommerce-aws-cloud-eks/services/shared-lib => ../shared-lib

require (
	github.com/Moiz-Ali-Moomin/microservice-ecommerce-aws-cloud-eks/services/shared-lib v0.0.0-00010101000000-000000000000
	go.uber.org/zap v1.27.0
)

require (
	github.com/stretchr/testify v1.11.1 // indirect
	go.uber.org/multierr v1.10.0 // indirect
)
