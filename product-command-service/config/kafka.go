package config

type KafkaConfig struct {
	BrokerAddress   string
	BrokerTopic     string
	BrokerPartition int
}
