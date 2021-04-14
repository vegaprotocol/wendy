package main

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/spf13/viper"
	cfg "github.com/tendermint/tendermint/config"
	"github.com/tendermint/tendermint/libs/log"
	"github.com/tendermint/tendermint/p2p"
	"github.com/tendermint/tendermint/p2p/conn"
	"github.com/tendermint/tendermint/privval"
	"github.com/tendermint/tendermint/proxy"

	"code.vegaprotocol.io/wendy/tendermint/app"
	nm "code.vegaprotocol.io/wendy/tendermint/node"
)

func newConfig(root string) *cfg.Config {
	viper.Set("home", root)
	viper.SetConfigName("config")
	viper.AddConfigPath(root)
	viper.AddConfigPath(filepath.Join(root, "config"))

	if err := viper.ReadInConfig(); err != nil {
		panic(err)
	}

	config := cfg.DefaultConfig()
	if err := viper.Unmarshal(config); err != nil {
		panic(err)
	}
	config.SetRoot(os.ExpandEnv(root))
	cfg.EnsureRoot(config.RootDir)

	return config
}

func main() {
	var root = "$HOME/.tendermint"
	if len(os.Args) > 1 {
		root = os.Args[1]
	}
	config := newConfig(root)
	nodeKey, err := p2p.LoadOrGenNodeKey(config.NodeKeyFile())
	if err != nil {
		panic(err)
	}

	filePV := privval.LoadOrGenFilePV(config.PrivValidatorKeyFile(), config.PrivValidatorStateFile())
	logger := log.NewTMLogger(log.NewSyncWriter(os.Stdout))

	app := app.New()

	node, err := nm.NewNode(
		config,
		filePV,
		nodeKey,
		proxy.NewLocalClientCreator(app),
		nm.DefaultGenesisDocProviderFunc(config),
		nm.DefaultDBProvider,
		nm.DefaultMetricsProvider(config.Instrumentation),
		log.NewFilter(logger,
			log.AllowDebugWith("module", "p2p"),
			log.AllowInfoWith("module", "app"),
			log.AllowInfoWith("module", "main"),
			log.AllowInfoWith("module", "state"),
			log.AllowError(),
		),
		nm.CustomReactors(map[string]p2p.Reactor{
			"TESTING": newReactor(),
		}),
	)
	if err != nil {
		panic(err)
	}

	node.Start()
	node.Wait()
}

type reactor struct {
	p2p.BaseReactor
}

func newReactor() *reactor {
	r := &reactor{}
	r.BaseReactor = *p2p.NewBaseReactor("TESTING", r)
	return r
}

func (*reactor) GetChannels() []*conn.ChannelDescriptor {
	return []*conn.ChannelDescriptor{
		{ID: 0x98, Priority: 4},
	}
}

func (*reactor) AddPeer(peer p2p.Peer) {
	go func() {
		for {
			time.Sleep(1 * time.Second)
			fmt.Printf("SENDING DATA\n")
			peer.Send(0x98, []byte("HELLO, WORLD!"))
		}
	}()
}

func (*reactor) Receive(chID byte, peer p2p.Peer, msgBytes []byte) {
	fmt.Printf("!!!!!!!!RECV DATA@%X: %s\n", chID, msgBytes)
}
