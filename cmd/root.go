/*
Copyright © 2024 mannk khacman98@gmail.com
*/
package cmd

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net"
	"os"
	"path/filepath"
	"time"

	"net/http"

	"github.com/mannk98/goutils/utils"
	log "github.com/sirupsen/logrus"
	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

type Container_infos struct {
	Index    int    `json:"index"`
	Upstream string `json:"upstream"`
	IpPort   string `json:"name"`
}

var (
	cfgFile  string
	Logger   = log.New()
	LogLevel = log.InfoLevel
	LogFile  = "healcheck_nginx.log"

	C      ConfigFile
	myhttp *http.Client
)

type ConfigFile struct {
	NginxContainerIP string
}

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "healcheck_nginx",
	Short: "Convert nginx-healcheck json to zabbix-json use for template discovery",
	Long:  ``,
	// Uncomment the following line if your bare application
	// has an action associated with it:
	Run: rootRun,
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func init() {
	//utils.InitLogger(LogFile, Logger, LogLevel)

	utils.InitLogger(LogFile, Logger, LogLevel)
	cobra.OnInitialize(initConfig)

	// Here you will define your flags and configuration settings.
	// Cobra supports persistent flags, which, if defined here,
	// will be global for your application.

	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.healcheck_nginx.toml)")

	// Cobra also supports local flags, which will only run
	// when this action is called directly.
	rootCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")

	//NginxContainerIP = viper.GetString(NginxContainerIP)
}

// initConfig reads in config file and ENV variables if set.
func initConfig() {
	//var home string
	var err error
	if cfgFile != "" {
		// Use config file from the flag.
		viper.SetConfigFile(cfgFile)
	} else {
		// Find home directory.
		/* 		home, err = os.UserHomeDir()
		   		if err != nil {
		   			Logger.Error(err)
		   			os.Exit(1)
		   		} */

		// Search config in home directory with name ".healcheck_nginx" (without extension).
		viper.AddConfigPath(".")
		//viper.AddConfigPath("./")
		viper.SetConfigType("toml")
		viper.SetConfigName(".healcheck_nginx.toml")
	}

	if !utils.PathIsExist(filepath.Join("./", ".healcheck_nginx.toml")) {
		nginx_ip := os.Getenv("NGINX_IP")
		if nginx_ip == "" {
			Logger.Error("nginx_ip variable not set or empty.")
		} else {
			Logger.Info("NGINX_IP:", nginx_ip)
			log.Info("NGINX_IP:", nginx_ip)
			//fmt.Println(home)
			_, err = utils.FileCreateWithContent(filepath.Join("./", ".healcheck_nginx.toml"), []byte("NginxContainerIP="+"'http://"+nginx_ip+"/status/bestatus'"))
			if err != nil {
				Logger.Error(err)
			}
		}
	}

	viper.AutomaticEnv() // read in environment variables that match

	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			Logger.Error(".healcheck_nginx.toml file at ./ folder is not exist. Creating it first before use")
		} else {
			Logger.Error(err)
		}
	}

	if err := viper.Unmarshal(&C); err != nil {
		Logger.Error("Error unmarshalling config: ", err)
		os.Exit(1)
	}

	customTransport := HttpClientNewTransPort()
	myhttp = &http.Client{
		Transport: customTransport,
	}
}

func rootRun(cmd *cobra.Command, args []string) {
	var zabbix_json string
	_, apihealthcheckJson, err := HttpGet(C.NginxContainerIP)
	if err != nil {
		Logger.Error(err)
		os.Exit(1)
	}
	//fmt.Println(apihealthcheckJson)

	result := gjson.Get(apihealthcheckJson, "servers.server")
	result.ForEach(func(key, value gjson.Result) bool {
		//println(value.String())
		index := gjson.Get(value.String(), "index")
		upstream := gjson.Get(value.String(), "upstream")
		name := gjson.Get(value.String(), "name")
		status := gjson.Get(value.String(), "status")
		//fmt.Println(index, upstream, name)

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
	//fmt.Println(zabbix_json)
}

func HttpClientNewTransPort() *http.Transport {
	return &http.Transport{
		Proxy: http.ProxyFromEnvironment,
		DialContext: (&net.Dialer{
			Timeout:   30 * time.Second,
			KeepAlive: 15 * time.Second,
			DualStack: true,
		}).DialContext,
		ForceAttemptHTTP2:   true,
		MaxIdleConns:        100,
		MaxIdleConnsPerHost: 100,
		IdleConnTimeout:     30 * time.Second,
		TLSHandshakeTimeout: 6 * time.Second,
		//		ExpectContinueTimeout:  1 * time.Second,
		MaxResponseHeaderBytes: 8192,
		ResponseHeaderTimeout:  time.Millisecond * 5000,
		DisableKeepAlives:      false,
	}
}

// Don't forget add https:// or http
func HttpGet(url string) (*http.Response, string, error) {
	response, err := myhttp.Get(url)
	if err != nil {
		//Logger.Error(err)
		return response, "", err
	}
	defer response.Body.Close()

	// Read the response body
	body, err := ioutil.ReadAll(response.Body)
	if err != nil {
		Logger.Error("Error reading response body:", err)
		return response, "", err
	}
	return response, string(body), err
}

func PrettyPrintJson(jsonStr string) (string, error) {
	var data interface{}
	err := json.Unmarshal([]byte(jsonStr), &data)
	if err != nil {
		return "", err
	}

	// Marshal the data with indentation
	prettyJSON, err := json.MarshalIndent(data, "", string(byte('\t')))
	if err != nil {
		return "", err
	}
	return string(prettyJSON), err
}
