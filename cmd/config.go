package cmd

import (
	"fmt"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	config "github.com/weldpua2008/supraworker/config"
)

// Init
func init() {
	// Init config file for viper
	rootCmd.PersistentFlags().StringVar(&config.CfgFile, "config", "", "config file (default is $HOME/supraworker.yaml)")
	viper.SetDefault("license", "apache")
	configCMD.PersistentFlags().Bool("viper", true, "use Viper for configuration")
	viper.Set("Verbose", true)
	rootCmd.AddCommand(configCMD)
}

// Pring config path command
var configCMD = &cobra.Command{
	Use: "configpath",
	Run: func(command *cobra.Command, args []string) {
		fmt.Println("Config file:", viper.ConfigFileUsed())
	},
}
