package main

// GinUser for gin
type GinUser struct {
	User   string `json:"user,omitempty" yaml:"user,omitempty"`
	Passwd string `json:"passwd,omitempty" yaml:"passwd,omitempty"`
}

// SDKConfig for fabric-sdk-go
type SDKConfig struct {
	ConfigPath    string   `json:"configPath,omitempty" yaml:"configPath,omitempty" `
	OrgName       string   `json:"orgName,omitempty" yaml:"orgName,omitempty" `
	UserName      string   `json:"userName,omitempty" yaml:"userName,omitempty" `
	TargetPeers   []string `json:"targetPeers,omitempty" yaml:"targetPeers,omitempty"`
	TargetOrderer string   `json:"targetOrderer,omitempty" yaml:"targetOrderer,omitempty"`
}

// Channel define channel info
type Channel struct {
	ChannelID   string   `json:"channelID,omitempty" yaml:"channelID,omitempty"`
	CCName      string   `json:"ccName,omitempty" yaml:"ccName,omitempty"`
	TargetPeers []string `json:"targetPeers,omitempty" yaml:"targetPeers,omitempty"`
}

// RestfulServer for server
type RestfulServer struct {
	Port     string    `json:"port,omitempty" yaml:"port,omitempty" `
	GinUsers []GinUser `json:"ginuser,omitempty" yaml:"ginuser,omitempty" `
}

// Chaincode define a chaincode
type Chaincode struct {
	ChaincodeID string `json:"chaincodeID,omitempty" yaml:"chaincodeID,omitempty"`
	Version     string `json:"version,omitempty" yaml:"version,omitempty"`
	Path        string `json:"path,omitempty" yaml:"path,omitempty"`
}

// ServerConfig for server
type ServerConfig struct {
	SDKConfig     `json:"sdkconfig,omitempty" yaml:"sdkconfig,omitempty"`
	RestfulServer `json:"restfulserver,omitempty" yaml:"restfulserver,omitempty"`
}

// Parameters define Parameters struct
type Parameters struct {
	ChannelID   string   `json:"channelID,omitempty" yaml:"channelID,omitempty"`
	ChaincodeID string   `json:"chaincodeID,omitempty" yaml:"chaincodeID,omitempty"`
	Function    string   `json:"function,omitempty" yaml:"function,omitempty"`
	Args        []string `json:"args,omitempty" yaml:"args,omitempty"`
}
