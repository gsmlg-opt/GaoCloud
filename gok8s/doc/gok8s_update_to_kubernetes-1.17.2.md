# gok8s update to kubernetes-1.17.2

## client/apiutil/apimachinery.go

结构体名字由 k8s.io/apimachinery/pkg/runtime/serializer.DirectCodecFactory 变成 k8s.io/apimachinery/pkg/runtime/serializer.WithoutConversionCodecFactory

## client/apiutil/memcache.go
k8s.io/client-go/discovery.CachedDiscoveryInterface 接口需要实现函数

func ServerGroupsAndResources() ([]*metav1.APIGroup, []*metav1.APIResourceList, error)

## client/unstructured_client.go
k8s.io/client-go/dynamic.ResourceInterface 接口函数Patch参数 metav1.UpdateOptions 变为 metav1.PatchOptions