package commands

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"path"
	"runtime"
	"strings"

	"github.com/bravetools/bravetools/platform"
	"github.com/bravetools/bravetools/shared"
	"github.com/spf13/cobra"
)

var hostInit = &cobra.Command{
	Use:   "init",
	Short: "Create a new Bravetools host",
	Long:  ``,
	Run:   serverInit,
}

var hostConfigPath, storage, ram, network, backendType string

func init() {
	includeInitFlags(hostInit)
}

func includeInitFlags(cmd *cobra.Command) {
	cmd.PersistentFlags().StringVarP(&hostConfigPath, "config", "", "", "Path to the host configuration file [OPTIONAL]")
	cmd.PersistentFlags().StringVarP(&storage, "storage", "s", "", "Host storage size [OPTIONAL]")
	cmd.PersistentFlags().StringVarP(&ram, "memory", "m", "", "Host memory size [OPTIONAL]")
	cmd.PersistentFlags().StringVarP(&network, "network", "n", "", "Host network IP range [OPTIONAL]")
	cmd.PersistentFlags().StringVarP(&backendType, "backend", "b", "", "Backend type (multipass or lxd) [OPTIONAL]")
}

func serverInit(cmd *cobra.Command, args []string) {
	userHome, _ := os.UserHomeDir()
	params := make(map[string]string)

	braveHome := false
	if _, err := os.Stat(path.Join(userHome, ".bravetools")); !os.IsNotExist(err) {
		braveHome = true
	}

	braveProfile := true
	remote := host.Remote
	_, err := platform.GetBraveProfile(remote)
	if err != nil {
		fmt.Println("Brave profile: ", err)
		braveProfile = false
	}
	if err == nil {
		braveProfile = true
	}

	if backendType == "" {
		hostOs := runtime.GOOS
		switch hostOs {
		case "linux":
			backendType = "lxd"
		case "darwin":
			backendType = "multipass"
		case "windows":
			backendType = "multipass"
		default:
			err := deleteBraveHome(userHome)
			if err != nil {
				log.Fatal(err.Error())
			}
			fmt.Println(runtime.GOOS)
			fmt.Println("Unsupported OS")
		}
	}

	if braveHome == false && braveProfile == false {
		err = createBraveHome(userHome)
		if err != nil {
			log.Fatal(err.Error())
		}

		if storage == "" {
			storage = "12"
		}
		params["storage"] = storage
		if ram == "" {
			ram = "4GB"
		}
		params["ram"] = ram
		if network == "" {
			network = "10.0.0.1"
		}
		params["network"] = network
		params["backend"] = backendType

		if hostConfigPath != "" {
			// TODO: validate configuration. Now assume that path ends with config.yml
			err = shared.CopyFile(hostConfigPath, path.Join(userHome, ".bravetools", "config.yml"))
			if err != nil {
				log.Fatal(err)
			}
			loadConfig()
		} else {
			userHome, _ := os.UserHomeDir()
			platform.SetupHostConfiguration(params, userHome)
			loadConfig()
		}

		err = backend.BraveBackendInit()
		if err != nil {
			fmt.Println("Error initializing Bravetools backend: ", err)
			log.Fatal(shared.REMOVELIN)
		}

		loadConfig()

		if backendType == "multipass" {
			info, err := backend.Info()

			if err != nil {
				log.Fatal(err)
			}

			settings := host.Settings
			settings.BackendSettings.Resources.IP = info.IPv4
			err = platform.UpdateBraveSettings(settings)

			if err != nil {
				log.Fatal(err)
			}

			loadConfig()
		}

		err = host.AddRemote()
		if err != nil {
			log.Fatal(err)
		}
	} else {
		scanner := bufio.NewScanner(os.Stdin)
		for {
			fmt.Println(shared.REINIT)
			scanner.Scan()
			in := scanner.Text()
			in = strings.ToLower(in)
			if in == "yes" || in == "y" {
				if backendType == "multipass" {
					log.Fatal(shared.REMOVEMP)
				} else {
					p := path.Join(userHome, ".bravetools/")

					if braveHome == false {
						log.Fatal(shared.REMOVELIN)
					} else {

						err1 := os.RemoveAll(p)
						err2 := platform.DeleteProfile(host.Settings.Profile, remote)
						err3 := platform.DeleteStoragePool(host.Settings.StoragePool.Name, remote)
						err4 := platform.DeleteNetwork("bravebr0", remote)

						if err1 != nil || err2 != nil || err3 != nil || err4 != nil {
							log.Fatal(shared.REMOVELIN)
						}
					}
				}

				break
			} else if in == "no" || in == "n" {
				break
			} else {
				continue
			}
		}
	}
}
