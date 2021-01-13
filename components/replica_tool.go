package components

type ReplicaTool struct {
	clusterIndexMap map[string]int32
}

func NewReplicaTool(clusterToolMap *map[string]*ClusterTool) ReplicaTool {
	clusterIndexMap := make(map[string]int32)
	i := 0
	for clusterId, _ := range *clusterToolMap {
		clusterIndexMap[clusterId] = int32(i)
		i++
	}
	return ReplicaTool{
		clusterIndexMap: clusterIndexMap,
	}
}

func (r *ReplicaTool) GetReplicas(clusterId string, totalReplicas *int32, replicaPolicy string) int32 {
	// Currently we only implement the basic replicaPolicy `avg`
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
