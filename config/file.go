package config

import (
	"fmt"
	os2 "os"
	"os/user"
	"strings"

	"github.com/cometbft/cometbft/libs/os"
	"github.com/tessellated-io/pickaxe/log"
)

// normalizeConfigFile loads a config file from a short path (ex. ~/.restake/config.yml => /home/tessellated/.restake/config.yaml)
func ReadFile(configFile string) (string, error) {
	expandedConfigFile := ExpandHomeDir(configFile)
	configOk := os.FileExists(expandedConfigFile)
	if !configOk {
		return "", fmt.Errorf("failed to load config file at: %s", configFile)
	}
	return expandedConfigFile, nil
}

func CreateDirectoryIfNeeded(configurationDirectory string, logger *log.Logger) error {
	expanded := ExpandHomeDir(configurationDirectory)
	exists, err := folderExists(expanded)
	if err != nil {
		return err
	}

	if exists {
		return nil
	}

	err = os2.MkdirAll(expanded, 0o755)
	if err != nil {
		return err
	}

	logger.Info().Str("configuration_dir", configurationDirectory).Msg("created configuration directory")

	return nil
}

func SafeWrite(file string, contents []byte, logger *log.Logger) error {
	expanded := ExpandHomeDir(file)
	if os.FileExists(expanded) {
		logger.Warn().Str("file", expanded).Msg("skipping overwriting existing file")
		return nil
	}

	err := os.WriteFile(expanded, contents, 0o644)
	if err != nil {
		return err
	}
	logger.Info().Str("file", expanded).Msg("wrote file")
	return nil
}

func ExpandHomeDir(path string) string {
	if !strings.HasPrefix(path, "~") {
		return path
	}

	usr, err := user.Current()
	if err != nil {
		panic(fmt.Errorf("failed to get user's home directory: %v", err))
	}
	return strings.Replace(path, "~", usr.HomeDir, 1)
}

func FileExists(filePath string) (bool, error) {
	expanded := ExpandHomeDir(filePath)
	exists := os.FileExists(expanded)

	return exists, nil
}

func folderExists(folderPath string) (bool, error) {
	fileInfo, err := os2.Stat(folderPath)
	if err != nil {
		if os2.IsNotExist(err) {
			// The folder does not exist
			return false, nil
		}
		// Some other error occurred when trying to access the folder
		return false, err
	}
	// Check if the path is indeed a folder/directory
	return fileInfo.IsDir(), nil
}
