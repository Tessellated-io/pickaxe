package chains

// Provides offline chain data for applications where we don't need everything under the sun
// TODO: This is kinda a hack and we should probably replace it with a cache / chain client
type OfflineChainRegistry struct {
	ChainIDToData       map[string]*ChainData
	ChainNameToData     map[string]*ChainData
	AccountPrefixToData map[string]*ChainData
}

func NewOfflineChainRegistry() *OfflineChainRegistry {
	chainRegistry := &OfflineChainRegistry{
		ChainIDToData:       make(map[string]*ChainData),
		ChainNameToData:     make(map[string]*ChainData),
		AccountPrefixToData: make(map[string]*ChainData),
	}

	chainRegistry.addToRegistry("axelar", "axelar-dojo-1", "axelar", "uaxl", 6, "axelar-validator.tessageo.net:9090")
	chainRegistry.addToRegistry("cosmoshub", "cosmoshub-4", "cosmos", "uatom", 6, "cosmos-validator.tessageo.net:9090")
	chainRegistry.addToRegistry("evmos", "evmos_9001-2", "evmos", "aevmos", 18, "evmos-validator.tessageo.net:9090")
	chainRegistry.addToRegistry("gravitybridge", "gravity-bridge-3", "gravity", "ugraviton", 6, "gravity-validator.tessageo.net:9090")
	chainRegistry.addToRegistry("juno", "juno-1", "juno", "ujuno", 6, "juno-validator.tessageo.net:9090")
	chainRegistry.addToRegistry("mars", "mars-1", "mars", "umars", 6, "mars-validator.tessageo.net:9090")
	chainRegistry.addToRegistry("neutron", "neutron-1", "neutron", "untrn", 6, "neutron-validator.tessageo.net:9090")
	chainRegistry.addToRegistry("osmosis", "osmosis-1", "osmo", "uosmo", 6, "osmosis-validator.tessageo.net:9090")
	chainRegistry.addToRegistry("sommelier", "sommelier-3", "somm", "usomm", 6, "sommelier-validator.tessageo.net:9090")
	chainRegistry.addToRegistry("stride", "stride-1", "stride", "ustrd", 6, "stride-validator.tessageo.net:9090")

	return chainRegistry
}

func (cr *OfflineChainRegistry) addToRegistry(
	chainName string,
	chainID string,
	accountPrefix string,
	nativeToken string,
	nativeTokenDecimals int,
	grpcUrl string,
) {
	chainData := &ChainData{
		ChainID:       chainID,
		ChainName:     chainName,
		AccountPrefix: accountPrefix,

		NativeToken:         nativeToken,
		NativeTokenDecimals: nativeTokenDecimals,

		GrpcUrl: grpcUrl,
	}

	cr.ChainNameToData[chainName] = chainData
	cr.ChainIDToData[chainID] = chainData
	cr.AccountPrefixToData[accountPrefix] = chainData
}
