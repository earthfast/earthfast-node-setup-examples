package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"golang.org/x/crypto/ssh"
	"gopkg.in/yaml.v2"
)

type Node struct {
	Host   string `yaml:"host"`
	SSHKey string `yaml:"ssh_key"`
}

type Config map[string]Node

var configFile string

var rootCmd = &cobra.Command{
	Use:     "earthfast-cnd",
	Short:   "Earthfast Content Node Deployer",
	Long:    "Manage multiple Earthfast content nodes across VMs",
	Version: "1.0.0",
}

func init() {
	rootCmd.PersistentFlags().StringVar(&configFile, "config", "~/.earthfast-cnd/config.yaml", "config file")

	// Add flags for the add command
	addCmd.Flags().String("name", "", "Name for the new node")
	addCmd.Flags().String("host", "", "Hostname or IP of the VM")
	addCmd.Flags().String("ssh-key", "", "Path to SSH key for the VM")

	// Mark flags as required
	addCmd.MarkFlagRequired("name")
	addCmd.MarkFlagRequired("host")
	addCmd.MarkFlagRequired("ssh-key")

	rootCmd.AddCommand(listCmd, addCmd, removeCmd, deployCmd, startCmd, stopCmd, logsCmd)
}

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List all managed Earthfast content nodes",
	Run: func(cmd *cobra.Command, args []string) {
		config := loadConfig()
		for name, node := range config {
			fmt.Printf("%s: %s (Status: %s)\n", name, node.Host, checkStatus(node))
		}
	},
}

var addCmd = &cobra.Command{
	Use:   "add",
	Short: "Add a new Earthfast content node",
	Run: func(cmd *cobra.Command, args []string) {
		name, _ := cmd.Flags().GetString("name")
		host, _ := cmd.Flags().GetString("host")
		sshKey, _ := cmd.Flags().GetString("ssh-key")

		config := loadConfig()
		config[name] = Node{Host: host, SSHKey: sshKey}
		saveConfig(config)
		fmt.Printf("Added node %s with host %s\n", name, host)
	},
}

var removeCmd = &cobra.Command{
	Use:   "remove [name]",
	Short: "Remove an Earthfast content node",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		name := args[0]
		config := loadConfig()
		if _, exists := config[name]; exists {
			delete(config, name)
			saveConfig(config)
			fmt.Printf("Removed node %s\n", name)
		} else {
			fmt.Printf("Node %s not found\n", name)
		}
	},
}

var deployCmd = &cobra.Command{
	Use:   "deploy [name]",
	Short: "Deploy an Earthfast content node",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		name := args[0]
		config := loadConfig()
		if node, exists := config[name]; exists {
			deployNode(node)
			fmt.Printf("Deployed node %s\n", name)
		} else {
			fmt.Printf("Node %s not found\n", name)
		}
	},
}

var startCmd = &cobra.Command{
	Use:   "start [name]",
	Short: "Start an Earthfast content node",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		name := args[0]
		config := loadConfig()
		if node, exists := config[name]; exists {
			startNode(node)
			fmt.Printf("Started node %s\n", name)
		} else {
			fmt.Printf("Node %s not found\n", name)
		}
	},
}

var stopCmd = &cobra.Command{
	Use:   "stop [name]",
	Short: "Stop an Earthfast content node",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		name := args[0]
		config := loadConfig()
		if node, exists := config[name]; exists {
			stopNode(node)
			fmt.Printf("Stopped node %s\n", name)
		} else {
			fmt.Printf("Node %s not found\n", name)
		}
	},
}

var logsCmd = &cobra.Command{
	Use:   "logs [name]",
	Short: "View logs for an Earthfast content node",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		name := args[0]
		config := loadConfig()
		if node, exists := config[name]; exists {
			viewLogs(node)
		} else {
			fmt.Printf("Node %s not found\n", name)
		}
	},
}

func loadConfig() Config {
	expandedPath, _ := filepath.Abs(configFile)
	data, err := ioutil.ReadFile(expandedPath)
	if err != nil {
		return make(Config)
	}

	var config Config
	err = yaml.Unmarshal(data, &config)
	if err != nil {
		log.Fatalf("error: %v", err)
	}
	return config
}

func saveConfig(config Config) {
	expandedPath, _ := filepath.Abs(configFile)
	data, err := yaml.Marshal(config)
	if err != nil {
		log.Fatalf("error: %v", err)
	}

	err = os.MkdirAll(filepath.Dir(expandedPath), os.ModePerm)
	if err != nil {
		log.Fatalf("error: %v", err)
	}

	err = ioutil.WriteFile(expandedPath, data, 0644)
	if err != nil {
		log.Fatalf("error: %v", err)
	}
}

func checkStatus(node Node) string {
	// Implement status checking logic here
	return "Unknown"
}

func deployNode(node Node) {
	// Implement node deployment logic here
}

func startNode(node Node) {
	runSSHCommand(node, "cd content-node/docker-compose && docker-compose up -d")
}

func stopNode(node Node) {
	runSSHCommand(node, "cd content-node/docker-compose && docker-compose down")
}

func viewLogs(node Node) {
	runSSHCommand(node, "cd content-node/docker-compose && docker-compose logs")
}

func runSSHCommand(node Node, command string) {
	key, err := ioutil.ReadFile(node.SSHKey)
	if err != nil {
		log.Fatalf("unable to read private key: %v", err)
	}

	signer, err := ssh.ParsePrivateKey(key)
	if err != nil {
		log.Fatalf("unable to parse private key: %v", err)
	}

	config := &ssh.ClientConfig{
		User: "root",
		Auth: []ssh.AuthMethod{
			ssh.PublicKeys(signer),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}

	client, err := ssh.Dial("tcp", node.Host+":22", config)
	if err != nil {
		log.Fatalf("unable to connect: %v", err)
	}
	defer client.Close()

	session, err := client.NewSession()
	if err != nil {
		log.Fatalf("unable to create session: %v", err)
	}
	defer session.Close()

	output, err := session.CombinedOutput(command)
	if err != nil {
		log.Fatalf("command execution failed: %v", err)
	}

	fmt.Println(string(output))
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
