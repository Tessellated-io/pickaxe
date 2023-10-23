package config

import (
	"fmt"
	"os/user"
	"strings"

	"github.com/cometbft/cometbft/libs/os"
)

// normalizeConfigFile loads a config file from a short path (ex. ~/.restake/config.yml => /home/tessellated/.restake/config.yaml)
func NormalizeConfigFile(configFile string) string {
	expandedConfigFile := expandHomeDir(configFile)
	configOk := os.FileExists(expandedConfigFile)
	if !configOk {
		panic(fmt.Sprintf("Failed to load config file at: %s", configFile))
	}
	return expandedConfigFile
}

func expandHomeDir(path string) string {
	if !strings.HasPrefix(path, "~") {
		return path
	}

	usr, err := user.Current()
	if err != nil {
		panic(fmt.Errorf("failed to get user's home directory: %v", err))
	}
	return strings.Replace(path, "~", usr.HomeDir, 1)
}
