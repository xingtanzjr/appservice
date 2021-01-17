package reconciler

import (
	"sort"

	components "metricsadvisor.ai/appservice/components"
)

type ReplicaTool struct {
	clusterIndexMap map[string]int32
}

func NewReplicaTool(clusterToolMap *map[string]*components.ClusterTool) ReplicaTool {
	var clusterIds []string
	clusterIndexMap := make(map[string]int32)

	for clusterId := range *clusterToolMap {
		clusterIds = append(clusterIds, clusterId)
	}
	// Becasue the map does not have sort ensurance,
	// sort the clusterIds to ensure that each clusterId will corresponding the fixed `Index` in clusterIndexMap
	sort.Strings(clusterIds)
	for idx, clusterId := range clusterIds {
		clusterIndexMap[clusterId] = int32(idx)
	}
	return ReplicaTool{
		clusterIndexMap: clusterIndexMap,
	}
}

func (r *ReplicaTool) GetReplicas(clusterId string, totalReplicas *int32, replicaPolicy string) int32 {
	// TODO: Currently we only implement the basic replicaPolicy `avg`
	clusterCount := len(r.clusterIndexMap)
	index := r.clusterIndexMap[clusterId]
	var plus int32
	if index < *totalReplicas%int32(clusterCount) {
		plus = 1
	} else {
		plus = 0
	}
	return *totalReplicas/int32(clusterCount) + plus
}
