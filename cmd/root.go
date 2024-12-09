/*
Copyright © 2024 mannk khacman98@gmail.com
*/
package cmd

import "C"
import (
	"fmt"
	"github.com/mannk98/goutils/utils"
	log "github.com/sirupsen/logrus"
	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"
	"net/http"
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var rootCmd = &cobra.Command{
	Use:   "healcheck_nginx",
	Short: "Convert nginx-healcheck json to zabbix-json use for template discovery",
	Long:  ``,
	Run:   rootRun,
}

func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func init() {
	utils.InitLogger(Global.LogPath, Logger, Global.Loglevel)
	cobra.OnInitialize(initConfig)

	//rootCmd.PersistentFlags().StringVar(&Global.ConfigPath, "config", "", "config file")
	rootCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}

// initConfig reads in config file and ENV variables if set.
func initConfig() {
	viper.AddConfigPath(".")
	viper.SetConfigType("toml")
	viper.SetConfigName(utils.PathBaseName(Global.ConfigPath))

	if !utils.PathIsExist(Global.ConfigPath) {
		nginx_ip := os.Getenv("NGINX_IP")
		if nginx_ip == "" {
			Logger.Error("NGINX_IP variable not set or empty. Exit.")
			os.Exit(1)
		} else {
			Logger.Info("NGINX_IP:", nginx_ip)
			log.Info("NGINX_IP:", nginx_ip)
			path, err := utils.FileCreateWithContent(Global.ConfigPath, []byte("NginxContainerIP="+"'http://"+nginx_ip+"/status/bestatus'"))
			if err != nil {
				Logger.Errorf("fail when create config file %s: %v", path, err)
				os.Exit(1)
			}
		}
	}
	viper.AutomaticEnv()

	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			Logger.Errorf("file %s  at ./ folder is not exist. Creating it first before use", Global.ConfigPath)
		} else {
			Logger.Errorf("error when read config file: %v", err)
		}
		os.Exit(1)
	}

	if err := viper.Unmarshal(&Global.Config); err != nil {
		Logger.Error("Error unmarshalling config: ", err)
		os.Exit(1)
	}

	Global.HttpClient = &http.Client{
		Transport: HttpClientNewTransPort(),
	}
}

func rootRun(cmd *cobra.Command, args []string) {
	var zabbix_json string
	_, apihealthcheckJson, err := HttpGet(Global.Config.NginxContainerIP)
	if err != nil {
		Logger.Error(err)
		os.Exit(1)
	}

	result := gjson.Get(apihealthcheckJson, "servers.server")
	result.ForEach(func(key, value gjson.Result) bool {
		index := gjson.Get(value.String(), "index")
		upstream := gjson.Get(value.String(), "upstream")
		name := gjson.Get(value.String(), "name")
		status := gjson.Get(value.String(), "status")

		if zabbix_json == "" {
			zabbix_json, _ = sjson.Set(``, "data.0", map[string]interface{}{"{#INDEX}": index.Int(), "{#UPSTREAM}": upstream.String(), "{#NAME}": name.String(), "{#STATUS} of " + name.String(): status.String()})
		} else {
			zabbix_json, _ = sjson.Set(zabbix_json, "data.-1", map[string]interface{}{"{#INDEX}": index.Int(), "{#UPSTREAM}": upstream.String(), "{#NAME}": name.String(), "{#STATUS} of " + name.String(): status.String()})
		}
		return true // keep iterating
	})

	// Print the pretty printed JSON string
	pretty, err := PrettyPrintJson(zabbix_json)
	if err != nil {
		Logger.Error(err)
	}
	fmt.Println(string(pretty))
}
