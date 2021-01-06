package main

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/hyperledger/fabric-sdk-go/pkg/client/channel"
	"github.com/hyperledger/fabric-sdk-go/pkg/fabsdk"
)

func hello(ctx *gin.Context) {
	ctx.JSON(http.StatusOK, gin.H{"response": string("hello")})
}

func parseParameters(ctx *gin.Context) {
	request = new(Parameters)
	if err := ctx.ShouldBindJSON(request); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	requestBytes, err := json.Marshal(request)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	log.Println("the received  Requestcategory--body is : ", string(requestBytes))
}

func createChannel(ctx *gin.Context) {
	// 解析参数
	parseParameters(ctx)
}

func joinChannel(ctx *gin.Context) {
	// 解析参数
	parseParameters(ctx)
}

func queryChannel(ctx *gin.Context) {
	// 解析参数
	parseParameters(ctx)
}

func createCC(ctx *gin.Context) {
	// 解析参数
	parseParameters(ctx)
}

func updateCC(ctx *gin.Context) {
	// 解析参数
	parseParameters(ctx)
}

func invokeCC(ctx *gin.Context) {
	// 解析参数
	parseParameters(ctx)

	log.Println(request.ChannelID, serverConfig.OrgName, serverConfig.UserName)

	channelContext := sdk.ChannelContext(request.ChannelID, fabsdk.WithOrg(serverConfig.OrgName), fabsdk.WithUser(serverConfig.UserName))
	client, err := channel.New(channelContext)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	args := make([][]byte, 0)
	for i, arg := range request.Args {
		log.Println("the invokeCC request.Arg : ", i, arg)
		args = append(args, []byte(arg))
	}

	result, err := InvokeCC(client,
		channel.Request{
			ChaincodeID: request.ChaincodeID,
			Fcn:         request.Function,
			Args:        args,
		},
	)
	if err != nil {
		log.Println("the invokeCC response err info is : ", err.Error())
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// result.Responses[0].Response.Payload
	status := result.Responses[0].Response.Status
	response := result.Responses[0].Response.Payload
	txid := result.TransactionID
	valid := result.TxValidationCode

	log.Println("the categorypost response is : ", txid, "====", valid, "===", string(response))
	ctx.JSON(http.StatusOK, gin.H{"status": status, "TxId": txid, "Valid": valid, "response": string(response)})
}

func queryCC(ctx *gin.Context) {
	// 解析参数
	parseParameters(ctx)

	channelContext := sdk.ChannelContext(request.ChannelID, fabsdk.WithUser(serverConfig.UserName), fabsdk.WithOrg(serverConfig.OrgName))
	client, err := channel.New(channelContext)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	args := make([][]byte, 0)
	for i, arg := range request.Args {
		log.Println("the queryCC request.Arg : ", i, arg)
		args = append(args, []byte(arg))
	}

	result, err := QueryCC(client,
		channel.Request{
			ChaincodeID: request.ChaincodeID,
			Fcn:         request.Function,
			Args:        args,
		},
	)
	if err != nil {
		log.Println("the queryCC response err info is : ", err.Error())
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	status := result.Responses[0].Response.Status
	response := result.Responses[0].Response.Payload

	log.Println("the response is : ", string(response))
	ctx.JSON(http.StatusOK, gin.H{"status": status, "response": string(response)})
}
