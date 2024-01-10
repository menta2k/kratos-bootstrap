package bootstrap

import (
	"os"
	"path/filepath"
	"strings"

	"google.golang.org/grpc"

	"github.com/go-kratos/kratos/v2/config"
	"github.com/go-kratos/kratos/v2/log"

	// file
	fileKratos "github.com/go-kratos/kratos/v2/config/file"

	// etcd
	etcdKratos "github.com/go-kratos/kratos/contrib/config/etcd/v2"
	etcdClient "go.etcd.io/etcd/client/v3"

	// consul
	consulKratos "github.com/go-kratos/kratos/contrib/config/consul/v2"
	consulApi "github.com/hashicorp/consul/api"

	// nacos
	nacosKratos "github.com/go-kratos/kratos/contrib/config/nacos/v2"
	nacosClients "github.com/nacos-group/nacos-sdk-go/clients"
	nacosConstant "github.com/nacos-group/nacos-sdk-go/common/constant"
	nacosVo "github.com/nacos-group/nacos-sdk-go/vo"

	// apollo
	apolloKratos "github.com/go-kratos/kratos/contrib/config/apollo/v2"

	// kubernetes
	k8sKratos "github.com/go-kratos/kratos/contrib/config/kubernetes/v2"
	k8sUtil "k8s.io/client-go/util/homedir"

	conf "github.com/menta2k/kratos-bootstrap/gen/api/go/conf/v1"
)

var commonConfig = &conf.Bootstrap{}
var configList []interface{}

const remoteConfigSourceConfigFile = "remote.yaml"

// RegisterConfig registration configuration
func RegisterConfig(c interface{}) {
	initBootstrapConfig()
	configList = append(configList, c)
}

func initBootstrapConfig() {
	if len(configList) > 0 {
		return
	}

	configList = append(configList, commonConfig)

	if commonConfig.Server == nil {
		commonConfig.Server = &conf.Server{}
		configList = append(configList, commonConfig.Server)
	}

	if commonConfig.Client == nil {
		commonConfig.Client = &conf.Client{}
		configList = append(configList, commonConfig.Client)
	}

	if commonConfig.Data == nil {
		commonConfig.Data = &conf.Data{}
		configList = append(configList, commonConfig.Data)
	}

	if commonConfig.Trace == nil {
		commonConfig.Trace = &conf.Tracer{}
		configList = append(configList, commonConfig.Trace)
	}

	if commonConfig.Logger == nil {
		commonConfig.Logger = &conf.Logger{}
		configList = append(configList, commonConfig.Logger)
	}

	if commonConfig.Registry == nil {
		commonConfig.Registry = &conf.Registry{}
		configList = append(configList, commonConfig.Registry)
	}

	if commonConfig.Oss == nil {
		commonConfig.Oss = &conf.OSS{}
		configList = append(configList, commonConfig.Oss)
	}

	if commonConfig.Notify == nil {
		commonConfig.Notify = &conf.Notification{}
		configList = append(configList, commonConfig.Notify)
	}
}

// NewConfigProvider creates a configuration
func NewConfigProvider(configPath string) config.Config {
	err, rc := LoadRemoteConfigSourceConfigs(configPath)
	if err != nil {
		log.Error("LoadRemoteConfigSourceConfigs: ", err.Error())
	}
	if rc != nil {
		return config.New(
			config.WithSource(
				NewFileConfigSource(configPath),
				NewRemoteConfigSource(rc),
			),
		)
	} else {
		return config.New(
			config.WithSource(
				NewFileConfigSource(configPath),
			),
		)
	}
}

// LoadBootstrapConfig loader boot configuration
func LoadBootstrapConfig(configPath string) error {
	cfg := NewConfigProvider(configPath)

	var err error

	if err = cfg.Load(); err != nil {
		return err
	}

	initBootstrapConfig()

	if err = scanConfigs(cfg); err != nil {
		return err
	}

	return nil
}

func scanConfigs(cfg config.Config) error {
	initBootstrapConfig()

	for _, c := range configList {
		if err := cfg.Scan(c); err != nil {
			return err
		}
	}
	return nil
}

func pathExists(path string) bool {
	_, err := os.Stat(path)
	if err == nil {
		return true
	}
	if os.IsNotExist(err) {
		return false
	}
	return false
}

// LoadRemoteConfigSourceConfigs loads the local configuration of the remote configuration source
func LoadRemoteConfigSourceConfigs(configPath string) (error, *conf.RemoteConfig) {
	configPath = configPath + "/" + remoteConfigSourceConfigFile
	if !pathExists(configPath) {
		return nil, nil
	}

	cfg := config.New(
		config.WithSource(
			NewFileConfigSource(configPath),
		),
	)
	defer func(cfg config.Config) {
		if err := cfg.Close(); err != nil {
			panic(err)
		}
	}(cfg)

	var err error

	if err = cfg.Load(); err != nil {
		return err, nil
	}

	if err = scanConfigs(cfg); err != nil {
		return err, nil
	}

	return nil, commonConfig.Config
}

type ConfigType string

const (
	ConfigTypeLocalFile  ConfigType = "file"
	ConfigTypeNacos      ConfigType = "nacos"
	ConfigTypeConsul     ConfigType = "consul"
	ConfigTypeEtcd       ConfigType = "etcd"
	ConfigTypeApollo     ConfigType = "apollo"
	ConfigTypeKubernetes ConfigType = "kubernetes"
	ConfigTypePolaris    ConfigType = "polaris"
)

// NewRemoteConfigSource creates a remote configuration source
func NewRemoteConfigSource(c *conf.RemoteConfig) config.Source {
	switch ConfigType(c.Type) {
	default:
		fallthrough
	case ConfigTypeLocalFile:
		return nil
	case ConfigTypeNacos:
		return NewNacosConfigSource(c)
	case ConfigTypeConsul:
		return NewConsulConfigSource(c)
	case ConfigTypeEtcd:
		return NewEtcdConfigSource(c)
	case ConfigTypeApollo:
		return NewApolloConfigSource(c)
	case ConfigTypeKubernetes:
		return NewKubernetesConfigSource(c)
	case ConfigTypePolaris:
		return NewPolarisConfigSource(c)
	}
}

// getConfigKey gets the legal configuration name
func getConfigKey(configKey string, useBackslash bool) string {
	if useBackslash {
		return strings.Replace(configKey, `.`, `/`, -1)
	} else {
		return configKey
	}
}

// NewFileConfigSource creates a local file configuration source
func NewFileConfigSource(filePath string) config.Source {
	return fileKratos.NewSource(filePath)
}

// NewNacosConfigSource creates a remote configuration source - Nacos
func NewNacosConfigSource(c *conf.RemoteConfig) config.Source {
	srvConf := []nacosConstant.ServerConfig{
		*nacosConstant.NewServerConfig(c.Nacos.Address, c.Nacos.Port),
	}

	cliConf := nacosConstant.ClientConfig{
		TimeoutMs:            10 * 1000, // http request timeout, in milliseconds
		BeatInterval:         5 * 1000,  // Heartbeat interval in milliseconds
		UpdateThreadNum:      20,        // Number of threads to update the service
		LogLevel:             "debug",
		CacheDir:             "../../configs/cache", // cache directory
		LogDir:               "../../configs/log",   // Log directory
		NotLoadCacheAtStart:  true,                  // Do not read local cache data at startup, true--do not read, false--read
		UpdateCacheWhenEmpty: true,                  // Whether to update the local cache when the service list is empty, true - update, false - not update
	}

	nacosClient, err := nacosClients.NewConfigClient(
		nacosVo.NacosClientParam{
			ClientConfig:  &cliConf,
			ServerConfigs: srvConf,
		},
	)
	if err != nil {
		log.Fatal(err)
	}

	return nacosKratos.NewConfigSource(nacosClient,
		nacosKratos.WithGroup(getConfigKey(c.Nacos.Key, false)),
		nacosKratos.WithDataID("bootstrap.yaml"),
	)
}

// NewEtcdConfigSource creates a remote configuration source - Etcd
func NewEtcdConfigSource(c *conf.RemoteConfig) config.Source {
	cfg := etcdClient.Config{
		Endpoints:   c.Etcd.Endpoints,
		DialTimeout: c.Etcd.Timeout.AsDuration(),
		DialOptions: []grpc.DialOption{grpc.WithBlock()},
	}

	cli, err := etcdClient.New(cfg)
	if err != nil {
		panic(err)
	}

	source, err := etcdKratos.New(cli, etcdKratos.WithPath(getConfigKey(c.Etcd.Key, true)))
	if err != nil {
		log.Fatal(err)
	}

	return source
}

// NewConsulConfigSource creates a remote configuration source - Consul
func NewConsulConfigSource(c *conf.RemoteConfig) config.Source {
	cfg := consulApi.DefaultConfig()
	cfg.Address = c.Consul.Address
	cfg.Scheme = c.Consul.Scheme

	cli, err := consulApi.NewClient(cfg)
	if err != nil {
		log.Fatal(err)
	}

	source, err := consulKratos.New(cli,
		consulKratos.WithPath(getConfigKey(c.Consul.Key, true)),
	)
	if err != nil {
		log.Fatal(err)
	}

	return source
}

// NewApolloConfigSource creates a remote configuration source - Apollo
func NewApolloConfigSource(c *conf.RemoteConfig) config.Source {
	source := apolloKratos.NewSource(
		apolloKratos.WithAppID(c.Apollo.AppId),
		apolloKratos.WithCluster(c.Apollo.Cluster),
		apolloKratos.WithEndpoint(c.Apollo.Endpoint),
		apolloKratos.WithNamespace(c.Apollo.Namespace),
		apolloKratos.WithSecret(c.Apollo.Secret),
		apolloKratos.WithEnableBackup(),
	)
	return source
}

// NewKubernetesConfigSource creates a remote configuration source - Kubernetes
func NewKubernetesConfigSource(c *conf.RemoteConfig) config.Source {
	source := k8sKratos.NewSource(
		k8sKratos.Namespace(c.Kubernetes.Namespace),
		k8sKratos.LabelSelector(""),
		k8sKratos.KubeConfig(filepath.Join(k8sUtil.HomeDir(), ".kube", "config")),
	)
	return source
}

// NewPolarisConfigSource creates a remote configuration source - Polaris
func NewPolarisConfigSource(_ *conf.RemoteConfig) config.Source {
	//configApi, err := polarisApi.NewConfigAPI()
	//if err != nil {
	//	log.Fatal(err)
	//}
	//
	//var opts []polarisKratos.Option
	//opts = append(opts, polarisKratos.WithNamespace("default"))
	//opts = append(opts, polarisKratos.WithFileGroup("default"))
	//opts = append(opts, polarisKratos.WithFileName("default.yaml"))
	//
	//source, err := polarisKratos.New(configApi, opts...)
	//if err != nil {
	//	log.Fatal(err)
	//}
	//
	//return source
	return nil
}
