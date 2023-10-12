package chains

type ChainData struct {
	ChainName     string
	ChainID       string
	AccountPrefix string

	GrpcUrl string

	NativeToken         string
	NativeTokenDecimals int
}
