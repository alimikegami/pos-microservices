# Point of Sales Microservices

This is a work-in-progress side project, serving as an experimental playground to help me explore the intricacies of microservices. Please note that some implementations of certain concepts may not be 100% correct, as this is primarily a learning exercise.

## Architecture Overview
The microservices in this project communicate via RESTful APIs and message queues.

## API Gateway
Kong (DB-less) is used as the API gateway for routing and load balancing.

## Services

### User Service
This service manages user accounts and authentication. It’s built with Go and PostgreSQL.

### Product Service
This service handles product and inventory management, implemented with Go, MongoDB, Elasticsearch, and Kafka. It follows the CQRS (Command Query Responsibility Segregation) pattern.

### Order Service
This service is responsible for order management and payment processing, integrated with the Midtrans payment gateway (currently in sandbox mode). It’s built with Go, PostgreSQL, and Kafka.
