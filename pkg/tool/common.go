package tool

import (
	"fmt"
	"reflect"
	"sort"
)

func IsSameList(foo []string, bar []string) bool {
	sort.Strings(foo)
	sort.Strings(bar)
	return reflect.DeepEqual(foo, bar)
}

func GetEndpointsString(servers []string, port int) string {
	sort.Strings(servers) // must be sorted for comparing
	endpoints := ""
	for _, value := range servers {
		endpoints += fmt.Sprintf("%s:%d;", value, port)
	}
	if len(endpoints) > 0 {
		endpoints = endpoints[:len(endpoints)-1]
	}
	return endpoints
}

func GetEndpointString(server string, port int) string {
	return fmt.Sprintf("%s:%d", server, port)
}
