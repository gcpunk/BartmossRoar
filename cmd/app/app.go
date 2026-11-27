package app

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/erashaperavm/BartmossRoar/core/api/configs"
	"github.com/erashaperavm/BartmossRoar/core/database"
	"github.com/erashaperavm/BartmossRoar/core/message"
	"github.com/erashaperavm/BartmossRoar/core/transport"
	"github.com/erashaperavm/BartmossRoar/core/utils"
	"github.com/spf13/cobra"
)

type App struct {
	cfg       Config
	db        database.DB
	dedupDB   database.DedupDB
	messageDB database.MessagesDB
	msgIDsDB  database.MsgIDsDB
	nodesDB   database.NodeDB
	potDB     database.PoTDB
	wd        string
	apiConfig configs.ApiConfig
	ts        []transport.Transport
}

func NewApp() *App {
	wd, err := os.Getwd()
	if err != nil {
		panic("can not get workdir")
	}

	// Load api config, br config
	brPath := path.Join(wd, "br.json")
	brCfg, err := ReadBrConfig(brPath)
	if err != nil {
		panic(err)
	}

	apiPath := path.Join(wd, "api_config.yaml")
	apiCfg, err := configs.ReadApiConfig(apiPath)
	if err != nil {
		panic(err)
	}

	var ts []transport.Transport
	for _, proto := range brCfg.Protocols {
		switch proto {
		case transport.ProtoToString(transport.TCP):
			for _, addr := range brCfg.AddressesIfTCP {
				t, err := transport.NewTCP(addr)
				if err != nil {
					panic(err)
				}
				ts = append(ts, t)
			}
		case transport.ProtoToString(transport.Tor):
			for _, port := range brCfg.LocalPortsIfTor {
				t, err := transport.NewTor(port, false) // do not use existing tor hidden service for now
				if err != nil {
					panic(err)
				}
				ts = append(ts, t)
			}
		default:
			panic("unsupported network protocol")
		}
	}

	return &App{
		wd:        wd,
		cfg:       brCfg,
		apiConfig: apiCfg,
		ts:        ts,
		// todo
	}
}

func (a *App) Start() error {
	// check there is badger db folder
	ok, err := utils.IsBadgerDir(a.wd)
	if err != nil {
		return err
	}
	if !ok {
		return fmt.Errorf("%s has no valid badger db. ", a.wd)
	}

	rootCmd := &cobra.Command{
		Use:   "br",
		Short: "BartmossRoar Core CLI tool. ",
		Long:  "Hidden your identity, chat free. ",
	}

	rootCmd.AddCommand(a.joinCmd())
	rootCmd.AddCommand(startCmd())

	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}

	return nil
}

func (a *App) joinCmd() *cobra.Command {
	return &cobra.Command{
		Use:     "join",
		Short:   "join a group",
		Long:    "join a bartmoss roar group",
		Example: "br join network.json",
		RunE: func(cmd *cobra.Command, args []string) error {
			// read the JSON
			networkJson := args[0]
			networkJsonBytes, err := os.ReadFile(networkJson)
			if err != nil {
				return err
			}

			// check if joined
			wd, err := os.Getwd()
			if err != nil {
				panic("can not get workdir")
			}
			networkID := strings.Trim(networkJson, ".json")
			pathToNtwk := path.Join(wd, "networks", networkID)
			if err := os.MkdirAll(filepath.Dir(pathToNtwk), 0750); err != nil {
				return fmt.Errorf("create parent dir: %w", err)
			}
			err = os.Mkdir(pathToNtwk, 0750)
			switch {
			case err == nil:
				// create dir successfully
			case errors.Is(err, fs.ErrExist):
				return errors.New("the network is joined")
			default:
				return fmt.Errorf("create network dir: %w", err)
			}

			// unmarshal
			nodeSetInfo, err := UnmarshalJSONToNodeSetInfo(networkJsonBytes)
			if err != nil {
				return err
			}

			// create connection with given nodes sequentially, if nobody response
			// it indicates a broken network or out-date network
			count := 0
			for _, node := range nodeSetInfo.Nodes {
				for _, nodeAddr := range node.Address {
					switch transport.WhichProtocol(nodeAddr) {
					case transport.TCP:
						for _, t := range a.ts {
							if t.Protocol() != transport.TCP {
								continue
							}
							// get a tcp transport instance
							err := t.Send(transport.NodeAddr{
								Protocol: transport.TCP,
								Address:  strings.Trim(nodeAddr, "tcp://"),
							}, message.OnlyTryPleaseIgnore)
							if err != nil {
								return err
							}
							// success
							count += 1
							break
						}
					case transport.Tor:
						for _, t := range a.ts {
							if t.Protocol() != transport.Tor {
								continue
							}
							// get a tor transport instance
							err := t.Send(transport.NodeAddr{
								Protocol: transport.Tor,
								Address:  nodeAddr,
							}, message.OnlyTryPleaseIgnore)
							if err != nil {
								return err
							}
							// success
							count += 1
							break
						}
					default:
						return errors.New("unsupported network protocol")
					}
				}
			}

			ratio := count / len(nodeSetInfo.Nodes)
			println("network %v success ratio: %v", networkJson, ratio)

			if count <= 2 {
				return errors.New("too few nodes")
			}

			// 有较多存活节点，载入数据库
			// 暂时无响应的也存入，因为可能是临时掉线
			// 可以后期使用 GC 清理
			db, dedup, messages, msgids, nodes, pot, err := database.Open(database.Options{
				Path: pathToNtwk,
			})
			if err != nil || db == nil {
				return nil
			}
			defer db.Close()

			a.dedupDB = dedup
			a.messageDB = messages
			a.msgIDsDB = msgids
			a.nodesDB = nodes
			a.potDB = pot

			for _, node := range nodeSetInfo.Nodes {
				err := a.nodesDB.RecordNode(database.NodeRecord{
					Identity:   database.ID(node.Identity),
					SignKey:    node.SignKey,
					EncryptKey: node.EncryptKey,
					Address:    node.Address,
				})
				if err != nil {
					return err
				}
			}

			return nil
		},
	}
}

func startCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "start",
		Short: "start the node",
		Long: `Start the BartmossRoar node.
			The node will:
				1. Discover neighbor nodes in the channel
				2. Sync and rotate ML-KEM public keys
				3. Tick dummy packets at 3-7s intervals
				4. Handle send, gossip and receive`,
		Example: "br start network_id",
		RunE: func(cmd *cobra.Command, args []string) error {

			return nil
		},
	}
}
