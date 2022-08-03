package tool

import "fmt"

func NameServiceName(rmqClusterName string) string {
	return fmt.Sprintf("%s-nameservice", rmqClusterName)
}

func BrokerName(rmqClusterName string, index int) string {
	return fmt.Sprintf("%s-broker-cluster-%d", rmqClusterName, index)
}

func ConsoleName(rmqClusterName string) string {
	return fmt.Sprintf("%s-console", rmqClusterName)
}
