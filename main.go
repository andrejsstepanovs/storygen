package main

import (
	"log"
	"os"

	"github.com/andrejsstepanovs/storygen/pkg"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func main() {
	initConfig()

	rootCmd := &cobra.Command{
		Use:   "main",
		Short: "Children Story Generator",
	}

	cmd, err := pkg.NewCommand()
	if err != nil {
		log.Fatalln(err)
	}
	rootCmd.AddCommand(
		cmd,
	)

	if err := rootCmd.Execute(); err != nil {
		log.Println(err)
		os.Exit(1)
	}
}

func initConfig() {
	//Step 1: Set the config file name and type
	viper.SetConfigName("app") // Name of the config file (without extension)
	viper.SetConfigType("env") // Type of the config file

	// Step 2: Add search paths for the config file
	// First, look in the current directory
	viper.AddConfigPath(".")

	// Fallback to the user's home directory
	home, err := os.UserHomeDir()
	if err != nil {
		log.Println("Error getting user home directory:", err)
		return
	}
	viper.AddConfigPath(home)

	// Step 3: Read the config file
	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			log.Println("Config file not found in current directory or home directory")
		} else {
			log.Println("Error reading config file:", err)
		}
		return
	}

	return
}
