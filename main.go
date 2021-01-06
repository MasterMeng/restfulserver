/*
This is an example for using fabric-sdk-go calling the fabcar chaincode in fabric-samples.
The fabric docker images and the fabric-samples used is v1.4.6.
Fabric-sdk-go is the latest.
*/

package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/hyperledger/fabric-sdk-go/pkg/core/config"
	"github.com/hyperledger/fabric-sdk-go/pkg/fabsdk"
	"gopkg.in/yaml.v2"
)

var (
	serverConfig *ServerConfig
	sdk          *fabsdk.FabricSDK
	request      *Parameters
)

func main() {
	var err error
	sdk, err = fabsdk.New(config.FromFile("./config/config-fabric.yaml"))
	if err != nil {
		fmt.Println(err)
		os.Exit(-1)
	}
	defer sdk.Close()

	buf, err := ioutil.ReadFile("./config/config-server.yaml")
	if err != nil {
		panic(err.Error())
	}
	serverConfig = new(ServerConfig)
	err = yaml.Unmarshal(buf, serverConfig)
	if err != nil {
		panic(err.Error())
	}
	serverBytes, _ := json.Marshal(serverConfig)
	fmt.Println(string(serverBytes))

	router := gin.Default()

	accounts := make(map[string]string, len(serverConfig.GinUsers))
	for _, account := range serverConfig.GinUsers {
		accounts[account.User] = account.Passwd
	}
	authorized := router.Group("/", gin.BasicAuth(accounts))

	authorized.POST("/hello", hello)
	authorized.POST("/channel/create", createChannel)
	authorized.POST("/channel/join", joinChannel)
	authorized.GET("/channel/query", queryChannel)
	authorized.POST("/cc/create", createCC)
	authorized.POST("/cc/invoke", invokeCC)
	authorized.POST("/cc/update", updateCC)
	authorized.GET("/cc/query", queryCC)

	router.Run(":" + serverConfig.Port)
}
