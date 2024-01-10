package bootstrap

import (
	"path/filepath"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/go-kratos/kratos/v2/registry"

	// etcd
	etcdKratos "github.com/go-kratos/kratos/contrib/registry/etcd/v2"
	etcdClient "go.etcd.io/etcd/client/v3"

	// consul
	consulKratos "github.com/go-kratos/kratos/contrib/registry/consul/v2"
	consulClient "github.com/hashicorp/consul/api"

	// eureka
	eurekaKratos "github.com/go-kratos/kratos/contrib/registry/eureka/v2"

	// nacos
	nacosKratos "github.com/go-kratos/kratos/contrib/registry/nacos/v2"
	nacosClients "github.com/nacos-group/nacos-sdk-go/clients"
	nacosConstant "github.com/nacos-group/nacos-sdk-go/common/constant"
	nacosVo "github.com/nacos-group/nacos-sdk-go/vo"

	// zookeeper
	zookeeperKratos "github.com/go-kratos/kratos/contrib/registry/zookeeper/v2"
	"github.com/go-zookeeper/zk"

	// kubernetes
	k8sRegistry "github.com/go-kratos/kratos/contrib/registry/kubernetes/v2"
	k8s "k8s.io/client-go/kubernetes"
	k8sRest "k8s.io/client-go/rest"
	k8sTools "k8s.io/client-go/tools/clientcmd"
	k8sUtil "k8s.io/client-go/util/homedir"

	// polaris
	//polarisKratos "github.com/go-kratos/kratos/contrib/registry/polaris/v2"

	// servicecomb
	servicecombClient "github.com/go-chassis/sc-client"
	servicecombKratos "github.com/go-kratos/kratos/contrib/registry/servicecomb/v2"

	conf "github.com/menta2k/kratos-bootstrap/gen/api/go/conf/v1"
)

type RegistryType string

const (
	RegistryTypeConsul    RegistryType = "consul"
	LoggerTypeEtcd        RegistryType = "etcd"
	LoggerTypeZooKeeper   RegistryType = "zookeeper"
	LoggerTypeNacos       RegistryType = "nacos"
	LoggerTypeKubernetes  RegistryType = "kubernetes"
	LoggerTypeEureka      RegistryType = "eureka"
	LoggerTypePolaris     RegistryType = "polaris"
	LoggerTypeServicecomb RegistryType = "servicecomb"
)

// NewRegistry creates a registration client
func NewRegistry(cfg *conf.Registry) registry.Registrar {
	if cfg == nil {
		return nil
	}

	switch RegistryType(cfg.Type) {
	case RegistryTypeConsul:
		return NewConsulRegistry(cfg)
	case LoggerTypeEtcd:
		return NewEtcdRegistry(cfg)
	case LoggerTypeZooKeeper:
		return NewZooKeeperRegistry(cfg)
	case LoggerTypeNacos:
		return NewNacosRegistry(cfg)
	case LoggerTypeKubernetes:
		return NewKubernetesRegistry(cfg)
	case LoggerTypeEureka:
		return NewEurekaRegistry(cfg)
	case LoggerTypePolaris:
		return nil
	case LoggerTypeServicecomb:
		return NewServicecombRegistry(cfg)
	}

	return nil
}

// NewDiscovery creates a discovery client
func NewDiscovery(cfg *conf.Registry) registry.Discovery {
	if cfg == nil {
		return nil
	}

	switch RegistryType(cfg.Type) {
	case RegistryTypeConsul:
		return NewConsulRegistry(cfg)
	case LoggerTypeEtcd:
		return NewEtcdRegistry(cfg)
	case LoggerTypeZooKeeper:
		return NewZooKeeperRegistry(cfg)
	case LoggerTypeNacos:
		return NewNacosRegistry(cfg)
	case LoggerTypeKubernetes:
		return NewKubernetesRegistry(cfg)
	case LoggerTypeEureka:
		return NewEurekaRegistry(cfg)
	case LoggerTypePolaris:
		return nil
	case LoggerTypeServicecomb:
		return NewServicecombRegistry(cfg)
	}

	return nil
}

// NewConsulRegistry creates a registration discovery client - Consul
func NewConsulRegistry(c *conf.Registry) *consulKratos.Registry {
	cfg := consulClient.DefaultConfig()
	cfg.Address = c.Consul.Address
	cfg.Scheme = c.Consul.Scheme

	var cli *consulClient.Client
	var err error
	if cli, err = consulClient.NewClient(cfg); err != nil {
		log.Fatal(err)
	}

	reg := consulKratos.New(cli, consulKratos.WithHealthCheck(c.Consul.HealthCheck))

	return reg
}

// NewEtcdRegistry creates a registration discovery client - Etcd
func NewEtcdRegistry(c *conf.Registry) *etcdKratos.Registry {
	cfg := etcdClient.Config{
		Endpoints: c.Etcd.Endpoints,
	}

	var err error
	var cli *etcdClient.Client
	if cli, err = etcdClient.New(cfg); err != nil {
		log.Fatal(err)
	}

	reg := etcdKratos.New(cli)

	return reg
}

// NewZooKeeperRegistry creates a registration discovery client - ZooKeeper
func NewZooKeeperRegistry(c *conf.Registry) *zookeeperKratos.Registry {
	conn, _, err := zk.Connect(c.Zookeeper.Endpoints, c.Zookeeper.Timeout.AsDuration())
	if err != nil {
		log.Fatal(err)
	}

	reg := zookeeperKratos.New(conn)
	if err != nil {
		log.Fatal(err)
	}

	return reg
}

// NewNacosRegistry creates a registration discovery client - Nacos
func NewNacosRegistry(c *conf.Registry) *nacosKratos.Registry {
	srvConf := []nacosConstant.ServerConfig{
		*nacosConstant.NewServerConfig(c.Nacos.Address, c.Nacos.Port),
	}

	cliConf := nacosConstant.ClientConfig{
		NamespaceId:          c.Nacos.NamespaceId,
		TimeoutMs:            uint64(c.Nacos.Timeout.AsDuration().Milliseconds()), // http request timeout, in milliseconds
		BeatInterval:         c.Nacos.BeatInterval.AsDuration().Milliseconds(),    //Heartbeat interval in milliseconds
		UpdateThreadNum:      int(c.Nacos.UpdateThreadNum),                        //Update the number of threads in the service
		LogLevel:             c.Nacos.LogLevel,
		CacheDir:             c.Nacos.CacheDir,             // cache directory
		LogDir:               c.Nacos.LogDir,               // Log directory
		NotLoadCacheAtStart:  c.Nacos.NotLoadCacheAtStart,  // Do not read local cache data at startup, true--do not read, false--read
		UpdateCacheWhenEmpty: c.Nacos.UpdateCacheWhenEmpty, // Whether to update the local cache when the service list is empty, true - update, false - not update
	}

	cli, err := nacosClients.NewNamingClient(
		nacosVo.NacosClientParam{
			ClientConfig:  &cliConf,
			ServerConfigs: srvConf,
		},
	)
	if err != nil {
		log.Fatal(err)
	}

	reg := nacosKratos.New(cli)

	return reg
}

// NewKubernetesRegistry creates a registration discovery client - Kubernetes
func NewKubernetesRegistry(_ *conf.Registry) *k8sRegistry.Registry {
	restConfig, err := k8sRest.InClusterConfig()
	if err != nil {
		home := k8sUtil.HomeDir()
		kubeConfig := filepath.Join(home, ".kube", "config")
		restConfig, err = k8sTools.BuildConfigFromFlags("", kubeConfig)
		if err != nil {
			log.Fatal(err)
			return nil
		}
	}

	clientSet, err := k8s.NewForConfig(restConfig)
	if err != nil {
		log.Fatal(err)
		return nil
	}

	reg := k8sRegistry.NewRegistry(clientSet)

	return reg
}

// NewEurekaRegistry creates a registration discovery client - Eureka
func NewEurekaRegistry(c *conf.Registry) *eurekaKratos.Registry {
	var opts []eurekaKratos.Option
	opts = append(opts, eurekaKratos.WithHeartbeat(c.Eureka.HeartbeatInterval.AsDuration()))
	opts = append(opts, eurekaKratos.WithRefresh(c.Eureka.RefreshInterval.AsDuration()))
	opts = append(opts, eurekaKratos.WithEurekaPath(c.Eureka.Path))

	var err error
	var reg *eurekaKratos.Registry
	if reg, err = eurekaKratos.New(c.Eureka.Endpoints, opts...); err != nil {
		log.Fatal(err)
	}

	return reg
}

// NewServicecombRegistry creates a registration discovery client - Servicecomb
func NewServicecombRegistry(c *conf.Registry) *servicecombKratos.Registry {
	cfg := servicecombClient.Options{
		Endpoints: c.Servicecomb.Endpoints,
	}

	var cli *servicecombClient.Client
	var err error
	if cli, err = servicecombClient.NewClient(cfg); err != nil {
		log.Fatal(err)
	}

	reg := servicecombKratos.NewRegistry(cli)

	return reg
}
