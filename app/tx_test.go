// nolint
package app

import (
	"encoding/hex"
	"fmt"
	"github.com/pokt-network/pocket-core/codec"
	types2 "github.com/pokt-network/pocket-core/codec/types"
	"github.com/pokt-network/pocket-core/crypto"
	"github.com/pokt-network/pocket-core/crypto/keys"
	sdk "github.com/pokt-network/pocket-core/types"
	apps "github.com/pokt-network/pocket-core/x/apps"
	appsTypes "github.com/pokt-network/pocket-core/x/apps/types"
	"github.com/pokt-network/pocket-core/x/auth"
	"github.com/pokt-network/pocket-core/x/auth/types"
	"github.com/pokt-network/pocket-core/x/gov"
	govTypes "github.com/pokt-network/pocket-core/x/gov/types"
	"github.com/pokt-network/pocket-core/x/nodes"
	nodeTypes "github.com/pokt-network/pocket-core/x/nodes/types"
	pocketTypes "github.com/pokt-network/pocket-core/x/pocketcore/types"
	"github.com/stretchr/testify/assert"
	rand2 "github.com/tendermint/tendermint/libs/rand"
	"github.com/tendermint/tendermint/node"
	tmTypes "github.com/tendermint/tendermint/types"
	"math/rand"
	"testing"
	"time"
)

func TestMain(m *testing.M) {
	pocketTypes.ClearSessionCache()
	pocketTypes.ClearEvidence()
	sdk.InitCtxCache(20)
	m.Run()
}

func TestS(t *testing.T) {
	bz, _ := hex.DecodeString("0a14da0d712e0ad5b37393c022c3333bed46c8667d0a1214b180d9f3f848a1133a7e63f31b09f932a2d3d9041a06313030303030")
	a := types2.Any{
		TypeUrl: "/x.nodes.MsgSend",
		Value:   bz,
	}
	cdc := memCodec()
	z := sdk.ProtoMsg(&nodeTypes.MsgSend{})
	cdc.ProtoCodec().UnpackAny(&a, &z)
	fmt.Println(z)
}

func TestZ(t *testing.T) {
	bz, _ := hex.DecodeString("0a14da0d712e0ad5b37393c022c3333bed46c8667d0a12146fc92bf55e9d9ddb3f320dafa7ca9b2a8fba59c91a06313030303030")
	s := nodeTypes.MsgSend{}
	memCodec().ProtoUnmarshalBinaryBare(bz, &s)
	fmt.Println(s)
}

func TestB(t *testing.T) {
	bz, _ := hex.DecodeString("da010a480a102f782e6e6f6465732e4d736753656e6412340a14da0d712e0ad5b37393c022c3333bed46c8667d0a12146fc92bf55e9d9ddb3f320dafa7ca9b2a8fba59c91a06313030303030120e0a0575706f6b74120531303030301a640a205bdbfdad5ab2f7fdd0b815cc0fee2e263242bcdb8db19db23765c3cc67815e12124049b5567c712541362bc74638ee18e1d28bdfb0247a90f066fdf0e293d4b1d56817256c33fb00a1a53e139765cbc560defd3235f95247048e0001942ad98cfc08220f54686973206973206120746573742128aa91c2cffe8db077")
	stdTx := types.ProtoStdTx{}
	err := memCodec().ProtoUnmarshalBinaryLengthPrefixed(bz, &stdTx)
	if err != nil {
		panic(err)
	}
	fmt.Println(hex.EncodeToString(stdTx.Msg.Value))
}

func Test(t *testing.T) {
	pkBz, _ := hex.DecodeString("49c249b58e8b6d240858c9c52755c1acf01d3f7b7df251e990fbd1879bd386275bdbfdad5ab2f7fdd0b815cc0fee2e263242bcdb8db19db23765c3cc67815e12")
	pk, _ := crypto.NewPrivateKeyBz(pkBz)
	addr1, _ := hex.DecodeString("da0d712e0ad5b37393c022c3333bed46c8667d0a")
	addr2, _ := hex.DecodeString("1fe307493be6804a1bcee853e0536298aa198325")
	tx := types.StdTx{
		Msg: nodeTypes.MsgSend{
			FromAddress: addr1,
			ToAddress:   addr2,
			Amount:      sdk.NewInt(100000),
		},
		Fee: sdk.Coins{sdk.Coin{Amount: sdk.NewInt(10000), Denom: "upokt"}},
		Signature: types.StdSignature{
			PublicKey: nil,
			Signature: nil,
		},
		Memo:    "This is a test!",
		Entropy: 67202631355893930,
	}
	sb, _ := auth.GetSignBytes("pocket-testet-playground", tx)
	fmt.Println(hex.EncodeToString(sb))
	res, err := pk.Sign(sb)
	if err != nil {
		panic(err)
	}
	tx.Signature.Signature = res
	pub := pk.PublicKey()
	fmt.Println(pub.VerifyBytes(sb, res))
	fmt.Println(hex.EncodeToString(res))
}

func TestUnstakeApp(t *testing.T) {
	tt := []struct {
		name         string
		memoryNodeFn func(t *testing.T, genesisState []byte) (tendermint *node.Node, keybase keys.Keybase, cleanup func())
		*upgrades
	}{
		{name: "unstake an amino app with amino codec", memoryNodeFn: NewInMemoryTendermintNodeAmino, upgrades: &upgrades{codecUpgrade{false, 7000}}},
		{name: "unstake a proto app with proto codec", memoryNodeFn: NewInMemoryTendermintNodeProto, upgrades: &upgrades{codecUpgrade{true, 2}}}, // todo: FULL PROTO SCENARIO
	}
	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			if tc.upgrades != nil { // NOTE: Use to perform neccesary upgrades for test
				codec.UpgradeHeight = tc.upgrades.codecUpgrade.height
				_ = memCodecMod(tc.upgrades.codecUpgrade.upgradeMod)
			}
			_, kb, cleanup := tc.memoryNodeFn(t, oneValTwoNodeGenesisState())
			time.Sleep(2 * time.Second)
			kp, err := kb.GetCoinbase()
			assert.Nil(t, err)
			_, _, evtChan := subscribeTo(t, tmTypes.EventNewBlock)
			var tx *sdk.TxResponse
			var chains = []string{"0001"}
			<-evtChan // Wait for block
			memCli, _, evtChan := subscribeTo(t, tmTypes.EventTx)
			tx, err = apps.StakeTx(memCodec(), memCli, kb, chains, sdk.NewInt(1000000), kp, "test", tc.codecUpgrade.upgradeMod)
			assert.Nil(t, err)
			assert.NotNil(t, tx)

			<-evtChan // Wait for tx
			got, err := PCA.QueryApps(PCA.LastBlockHeight(), appsTypes.QueryApplicationsWithOpts{
				Page:  1,
				Limit: 1})
			assert.Nil(t, err)
			res := got.Result.(appsTypes.Applications)
			assert.Equal(t, 1, len(res))
			memCli, stopCli, evtChan := subscribeTo(t, tmTypes.EventTx)
			_, _ = apps.UnstakeTx(memCodec(), memCli, kb, kp.GetAddress(), "test", tc.codecUpgrade.upgradeMod)

			<-evtChan // Wait for tx
			got, err = PCA.QueryApps(PCA.LastBlockHeight(), appsTypes.QueryApplicationsWithOpts{
				Page:          1,
				Limit:         1,
				StakingStatus: 1,
			})
			assert.Nil(t, err)
			res = got.Result.(appsTypes.Applications)
			assert.Equal(t, 1, len(res))
			got, err = PCA.QueryApps(PCA.LastBlockHeight(), appsTypes.QueryApplicationsWithOpts{
				Page:          1,
				Limit:         1,
				StakingStatus: 2,
			})
			assert.Nil(t, err)
			res = got.Result.(appsTypes.Applications)
			assert.Equal(t, 1, len(res)) // default genesis application

			cleanup()
			stopCli()
		})
	}
}

func TestUnstakeNode(t *testing.T) {
	tt := []struct {
		name         string
		memoryNodeFn func(t *testing.T, genesisState []byte) (tendermint *node.Node, keybase keys.Keybase, cleanup func())
		*upgrades
	}{
		{name: "unstake a proto node with proto codec", memoryNodeFn: NewInMemoryTendermintNodeProto, upgrades: &upgrades{codecUpgrade{true, 2}}}, // TODO: PROTO FULL SCENARIO
		{name: "unstake an amino node with amino codec", memoryNodeFn: NewInMemoryTendermintNodeAmino, upgrades: &upgrades{codecUpgrade{false, 7000}}},
	}
	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			if tc.upgrades != nil { // NOTE: Use to perform neccesary upgrades for test
				codec.UpgradeHeight = tc.upgrades.codecUpgrade.height
				_ = memCodecMod(tc.upgrades.codecUpgrade.upgradeMod)
			}
			var chains = []string{"0001"}
			_, kb, cleanup := tc.memoryNodeFn(t, twoValTwoNodeGenesisState())
			time.Sleep(2 * time.Second)
			kp, err := kb.GetCoinbase()
			assert.Nil(t, err)
			_, _, evtChan := subscribeTo(t, tmTypes.EventNewBlock)
			var tx *sdk.TxResponse
			<-evtChan // Wait for block
			memCli, _, evtChan := subscribeTo(t, tmTypes.EventTx)
			_, err = PCA.QueryBalance(kp.GetAddress().String(), PCA.LastBlockHeight())
			assert.Nil(t, err)
			tx, err = nodes.UnstakeTx(memCodec(), memCli, kb, kp.GetAddress(), "test", tc.upgrades.upgradeMod)
			assert.Nil(t, err)
			assert.NotNil(t, tx)
			<-evtChan // Wait for tx
			_, _, evtChan = subscribeTo(t, tmTypes.EventNewBlockHeader)
			for {
				select {
				case res := <-evtChan:
					fmt.Println("HERE! ", res.Events)
					if len(res.Events["begin_unstake.module"]) == 1 {
						got, err := PCA.QueryNodes(PCA.LastBlockHeight(), nodeTypes.QueryValidatorsParams{StakingStatus: 1, JailedStatus: 0, Blockchain: "", Page: 1, Limit: 1}) // unstaking
						assert.Nil(t, err)
						res := got.Result.([]nodeTypes.Validator)
						assert.Equal(t, 1, len(res))
						got, err = PCA.QueryNodes(PCA.LastBlockHeight(), nodeTypes.QueryValidatorsParams{StakingStatus: 2, JailedStatus: 0, Blockchain: "", Page: 1, Limit: 1}) // staked
						assert.Nil(t, err)
						res = got.Result.([]nodeTypes.Validator)
						assert.Equal(t, 1, len(res))
						memCli, stopCli, evtChan := subscribeTo(t, tmTypes.EventNewBlockHeader)
						header := <-evtChan // Wait for header
						if len(header.Events["unstake.module"]) == 1 {
							got, err := PCA.QueryNodes(PCA.LastBlockHeight(), nodeTypes.QueryValidatorsParams{StakingStatus: 0, JailedStatus: 0, Blockchain: "", Page: 1, Limit: 1})
							assert.Nil(t, err)
							res := got.Result.([]nodeTypes.Validator)
							assert.Equal(t, 1, len(res))
							vals := got.Result.([]nodeTypes.Validator)
							addr := vals[0].Address
							balance, err := PCA.QueryBalance(addr.String(), PCA.LastBlockHeight())
							assert.Nil(t, err)
							assert.NotEqual(t, balance, sdk.ZeroInt())
							tx, err = nodes.StakeTx(memCodec(), memCli, kb, chains, "https://myPocketNode.com:8080", sdk.NewInt(10000000), kp, "test", tc.upgrades.upgradeMod)
							assert.Nil(t, err)
							assert.NotNil(t, tx)
							assert.Equal(t, tx.Code, uint32(0x0))
							cleanup()
							stopCli()

						}
						return
					}
				default:
					continue
				}
			}
		})
	}

}

func TestStakeNode(t *testing.T) {
	tt := []struct {
		name         string
		memoryNodeFn func(t *testing.T, genesisState []byte) (tendermint *node.Node, keybase keys.Keybase, cleanup func())
		*upgrades
	}{
		{name: "stake node with amino codec", memoryNodeFn: NewInMemoryTendermintNodeAmino, upgrades: &upgrades{codecUpgrade{false, 7000}}},
		{name: "stake a proto node with proto codec", memoryNodeFn: NewInMemoryTendermintNodeProto, upgrades: &upgrades{codecUpgrade{true, 2}}}, // TODO: FULL PROTO SCENARIO
	}
	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			if tc.upgrades != nil { // NOTE: Use to perform neccesary upgrades for test
				codec.UpgradeHeight = tc.upgrades.codecUpgrade.height
				_ = memCodecMod(tc.upgrades.codecUpgrade.upgradeMod)
			}
			_, kb, cleanup := tc.memoryNodeFn(t, twoValTwoNodeGenesisState())
			time.Sleep(2 * time.Second)
			kp, err := kb.GetCoinbase()
			assert.Nil(t, err)
			_, _, evtChan := subscribeTo(t, tmTypes.EventNewBlock)
			var tx *sdk.TxResponse
			var chains = []string{"0001"}
			<-evtChan // Wait for block
			memCli, stopCli, _ := subscribeTo(t, tmTypes.EventTx)
			tx, err = nodes.StakeTx(memCodec(), memCli, kb, chains, "https://myPocketNode.com:8080", sdk.NewInt(10000000), kp, "test", tc.upgrades.upgradeMod)
			assert.Nil(t, err)
			assert.NotNil(t, tx)
			assert.Equal(t, int(tx.Code), 0)
			cleanup()
			stopCli()

		})
	}
}

func TestStakeApp(t *testing.T) {
	tt := []struct {
		name         string
		memoryNodeFn func(t *testing.T, genesisState []byte) (tendermint *node.Node, keybase keys.Keybase, cleanup func())
		*upgrades
	}{
		{name: "stake app with amino codec", memoryNodeFn: NewInMemoryTendermintNodeAmino, upgrades: &upgrades{codecUpgrade{false, 7000}}},
		{name: "stake a proto app with proto codec", memoryNodeFn: NewInMemoryTendermintNodeProto, upgrades: &upgrades{codecUpgrade{true, 2}}}, // TODO FULL PROTO SCENARIO
	}
	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			if tc.upgrades != nil { // NOTE: Use to perform neccesary upgrades for test
				codec.UpgradeHeight = tc.upgrades.codecUpgrade.height
				_ = memCodecMod(tc.upgrades.codecUpgrade.upgradeMod)
			}

			_, kb, cleanup := tc.memoryNodeFn(t, oneValTwoNodeGenesisState())
			time.Sleep(2 * time.Second)
			kp, err := kb.GetCoinbase()
			assert.Nil(t, err)
			_, _, evtChan := subscribeTo(t, tmTypes.EventNewBlock)
			var tx *sdk.TxResponse
			var chains = []string{"0001"}

			<-evtChan // Wait for block
			memCli, stopCli, evtChan := subscribeTo(t, tmTypes.EventTx)
			tx, err = apps.StakeTx(memCodec(), memCli, kb, chains, sdk.NewInt(1000000), kp, "test", tc.codecUpgrade.upgradeMod)
			assert.Nil(t, err)
			assert.NotNil(t, tx)

			<-evtChan // Wait for tx
			got, err := PCA.QueryApps(PCA.LastBlockHeight(), appsTypes.QueryApplicationsWithOpts{
				Page:  1,
				Limit: 2,
			})
			assert.Nil(t, err)
			res := got.Result.(appsTypes.Applications)
			assert.Equal(t, 2, len(res))

			stopCli()
			cleanup()
		})
	}
}

func TestSendTransaction(t *testing.T) {

	tt := []struct {
		name         string
		memoryNodeFn func(t *testing.T, genesisState []byte) (tendermint *node.Node, keybase keys.Keybase, cleanup func())
		*upgrades
	}{
		{name: "send tx from an amino account with amino codec", memoryNodeFn: NewInMemoryTendermintNodeAmino, upgrades: &upgrades{codecUpgrade{false, 7000}}},
		{name: "send tx from a proto account with proto codec", memoryNodeFn: NewInMemoryTendermintNodeProto, upgrades: &upgrades{codecUpgrade{true, 2}}},
	}
	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			if tc.upgrades != nil { // NOTE: Use to perform neccesary upgrades for test
				codec.UpgradeHeight = tc.upgrades.codecUpgrade.height
				_ = memCodecMod(tc.upgrades.codecUpgrade.upgradeMod)
			}
			_, kb, cleanup := tc.memoryNodeFn(t, oneValTwoNodeGenesisState())
			time.Sleep(2 * time.Second)
			cb, err := kb.GetCoinbase()
			assert.Nil(t, err)
			kp, err := kb.Create("test")
			assert.Nil(t, err)
			_, _, evtChan := subscribeTo(t, tmTypes.EventNewBlock)
			var transferAmount = sdk.NewInt(1000)
			var tx *sdk.TxResponse

			<-evtChan // Wait for block
			memCli, stopCli, evtChan := subscribeTo(t, tmTypes.EventTx)
			tx, err = nodes.Send(memCodec(), memCli, kb, cb.GetAddress(), kp.GetAddress(), "test", transferAmount, tc.upgrades.upgradeMod)
			assert.Nil(t, err)
			assert.NotNil(t, tx)
			assert.Equal(t, int(tx.Code), 0)

			<-evtChan // Wait for tx
			balance, err := PCA.QueryBalance(kp.GetAddress().String(), PCA.LastBlockHeight())
			assert.Nil(t, err)
			assert.True(t, balance.Equal(transferAmount))
			balance, err = PCA.QueryBalance(cb.GetAddress().String(), PCA.LastBlockHeight())
			assert.Nil(t, err)

			cleanup()
			stopCli()
		})
	}
}

func TestDuplicateTxWithRawTx(t *testing.T) {
	tt := []struct {
		name         string
		memoryNodeFn func(t *testing.T, genesisState []byte) (tendermint *node.Node, keybase keys.Keybase, cleanup func())
		*upgrades
	}{
		{name: "send duplicate tx from an amino account with amino codec", memoryNodeFn: NewInMemoryTendermintNodeAmino, upgrades: &upgrades{codecUpgrade{false, 7000}}},
		{name: "send duplicate tx from a proto account with proto codec", memoryNodeFn: NewInMemoryTendermintNodeProto, upgrades: &upgrades{codecUpgrade{true, 2}}}, // TODO:  FULL PROTO SCENARIO
	}
	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			if tc.upgrades != nil { // NOTE: Use to perform neccesary upgrades for test
				codec.UpgradeHeight = tc.upgrades.codecUpgrade.height
				_ = memCodecMod(tc.upgrades.codecUpgrade.upgradeMod)
			}
			_, kb, cleanup := tc.memoryNodeFn(t, oneValTwoNodeGenesisState())
			time.Sleep(2 * time.Second)
			cb, err := kb.GetCoinbase()
			assert.Nil(t, err)
			kp, err := kb.Create("test")
			assert.Nil(t, err)
			pk, err := kb.ExportPrivateKeyObject(cb.GetAddress(), "test")
			assert.Nil(t, err)
			_, _, evtChan := subscribeTo(t, tmTypes.EventNewBlock)
			// create the transaction
			txBz, err := types.DefaultTxEncoder(memCodec())(types.NewTestTx(sdk.Context{}.WithChainID("pocket-test"),
				&nodeTypes.MsgSend{
					FromAddress: cb.GetAddress(),
					ToAddress:   kp.GetAddress(),
					Amount:      sdk.NewInt(1),
				},
				pk,
				rand2.Int64(),
				sdk.NewCoins(sdk.NewCoin(sdk.DefaultStakeDenom, sdk.NewInt(100000)))), -1)
			assert.Nil(t, err)
			// create the transaction
			_, err = types.DefaultTxEncoder(memCodec())(types.NewTestTx(sdk.Context{}.WithChainID("pocket-test"),
				&nodeTypes.MsgSend{
					FromAddress: cb.GetAddress(),
					ToAddress:   kp.GetAddress(),
					Amount:      sdk.NewInt(1),
				},
				pk,
				rand2.Int64(),
				sdk.NewCoins(sdk.NewCoin(sdk.DefaultStakeDenom, sdk.NewInt(100000)))), -1)
			assert.Nil(t, err)

			<-evtChan // Wait for block
			memCli, _, evtChan := subscribeTo(t, tmTypes.EventTx)
			_, err = nodes.RawTx(memCodec(), memCli, cb.GetAddress(), txBz)
			assert.Nil(t, err)
			// next tx
			<-evtChan // Wait for tx
			_, _, evtChan = subscribeTo(t, tmTypes.EventNewBlock)
			<-evtChan // Wait for  block
			memCli, stopCli, _ := subscribeTo(t, tmTypes.EventTx)
			txResp, err := nodes.RawTx(memCodec(), memCli, cb.GetAddress(), txBz)
			if err == nil && txResp.Code == 0 {
				t.Fatal("should fail on replay attack")
			}
			cleanup()
			stopCli()
		})
	}

}

func TestChangeParamsComplexTypeTx(t *testing.T) {
	tt := []struct {
		name         string
		memoryNodeFn func(t *testing.T, genesisState []byte) (tendermint *node.Node, keybase keys.Keybase, cleanup func())
		*upgrades
	}{
		{name: "change complex type params from an amino account with amino codec", memoryNodeFn: NewInMemoryTendermintNodeAmino, upgrades: &upgrades{codecUpgrade{false, 7000}}},
		{name: "change complex type params from a proto account with proto codec", memoryNodeFn: NewInMemoryTendermintNodeProto, upgrades: &upgrades{codecUpgrade{true, 2}}}, // TODO: FIX !!
	}
	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			if tc.upgrades != nil { // NOTE: Use to perform neccesary upgrades for test
				codec.UpgradeHeight = tc.upgrades.codecUpgrade.height
				_ = memCodecMod(tc.upgrades.codecUpgrade.upgradeMod)
			}
			resetTestACL()
			_, kb, cleanup := tc.memoryNodeFn(t, oneValTwoNodeGenesisState())
			time.Sleep(2 * time.Second)
			cb, err := kb.GetCoinbase()
			assert.Nil(t, err)
			kps, err := kb.List()
			assert.Nil(t, err)
			kp2 := kps[1]
			_, _, evtChan := subscribeTo(t, tmTypes.EventNewBlock)
			<-evtChan // Wait for block
			a := testACL
			a.SetOwner("gov/acl", kp2.GetAddress())
			memCli, stopCli, evtChan := subscribeTo(t, tmTypes.EventTx)
			tx, err := gov.ChangeParamsTx(memCodec(), memCli, kb, cb.GetAddress(), "gov/acl", a, "test", 1000000, false)
			assert.Nil(t, err)
			assert.NotNil(t, tx)
			select {
			case _ = <-evtChan:
				//fmt.Println(res)
				acl, err := PCA.QueryACL(PCA.LastBlockHeight())
				assert.Nil(t, err)
				o := acl.GetOwner("gov/acl")
				assert.Equal(t, kp2.GetAddress().String(), o.String())
				cleanup()
				stopCli()
			}
		})
	}
}

func TestChangeParamsSimpleTx(t *testing.T) {

	tt := []struct {
		name         string
		memoryNodeFn func(t *testing.T, genesisState []byte) (tendermint *node.Node, keybase keys.Keybase, cleanup func())
		*upgrades
	}{
		{name: "change complex type params from an amino account with amino codec", memoryNodeFn: NewInMemoryTendermintNodeAmino, upgrades: &upgrades{codecUpgrade{false, 7000}}},
		{name: "change complex type params from a proto account with proto codec", memoryNodeFn: NewInMemoryTendermintNodeProto, upgrades: &upgrades{codecUpgrade{true, 2}}}, // TODO: FULL PROTO SCENARIO
	}
	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			if tc.upgrades != nil { // NOTE: Use to perform neccesary upgrades for test
				codec.UpgradeHeight = tc.upgrades.codecUpgrade.height
				_ = memCodecMod(tc.upgrades.codecUpgrade.upgradeMod)
			}
			resetTestACL()
			_, kb, cleanup := tc.memoryNodeFn(t, oneValTwoNodeGenesisState())
			time.Sleep(2 * time.Second)
			cb, err := kb.GetCoinbase()
			assert.Nil(t, err)
			_, err = kb.List()
			assert.Nil(t, err)
			_, _, evtChan := subscribeTo(t, tmTypes.EventNewBlock)
			<-evtChan // Wait for block
			memCli, stopCli, evtChan := subscribeTo(t, tmTypes.EventTx)
			tx, err := gov.ChangeParamsTx(memCodec(), memCli, kb, cb.GetAddress(), "application/StabilityAdjustment", 100, "test", 1000000, false)
			assert.Nil(t, err)
			assert.NotNil(t, tx)
			select {
			case _ = <-evtChan:
				//fmt.Println(res)
				assert.Nil(t, err)
				o, _ := PCA.QueryParam(PCA.LastBlockHeight(), "application/StabilityAdjustment")
				assert.Equal(t, "100", o.Value)
				cleanup()
				stopCli()
			}
		})
	}
}

func TestUpgrade(t *testing.T) {
	tt := []struct {
		name         string
		memoryNodeFn func(t *testing.T, genesisState []byte) (tendermint *node.Node, keybase keys.Keybase, cleanup func())
		*upgrades
	}{
		{name: "change complex type params from an amino account with amino codec", memoryNodeFn: NewInMemoryTendermintNodeAmino, upgrades: &upgrades{codecUpgrade{false, 7000}}},
		{name: "change complex type params from a proto account with proto codec", memoryNodeFn: NewInMemoryTendermintNodeProto, upgrades: &upgrades{codecUpgrade{true, 2}}},
	}
	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			if tc.upgrades != nil { // NOTE: Use to perform neccesary upgrades for test
				codec.UpgradeHeight = tc.upgrades.codecUpgrade.height
				_ = memCodecMod(tc.upgrades.codecUpgrade.upgradeMod)
			}
			_, kb, cleanup := tc.memoryNodeFn(t, oneValTwoNodeGenesisState())
			time.Sleep(2 * time.Second)
			cb, err := kb.GetCoinbase()
			assert.Nil(t, err)
			_, _, evtChan := subscribeTo(t, tmTypes.EventNewBlock)
			var tx *sdk.TxResponse
			<-evtChan // Wait for block
			memCli, stopCli, evtChan := subscribeTo(t, tmTypes.EventTx)
			tx, err = gov.UpgradeTx(memCodec(), memCli, kb, cb.GetAddress(), govTypes.Upgrade{
				Height:  1000,
				Version: "2.0.0",
			}, "test", 1000000, tc.codecUpgrade.upgradeMod)
			assert.Nil(t, err)
			assert.NotNil(t, tx)

			<-evtChan // Wait for tx
			u, err := PCA.QueryUpgrade(PCA.LastBlockHeight())
			assert.Nil(t, err)
			assert.True(t, u.UpgradeVersion() == "2.0.0")

			cleanup()
			stopCli()
		})
	}
}

func TestDAOTransfer(t *testing.T) {
	tt := []struct {
		name         string
		memoryNodeFn func(t *testing.T, genesisState []byte) (tendermint *node.Node, keybase keys.Keybase, cleanup func())
		*upgrades
	}{
		{name: "change complex type params from an amino account with amino codec", memoryNodeFn: NewInMemoryTendermintNodeAmino, upgrades: &upgrades{codecUpgrade{false, 7000}}},
		{name: "change complex type params from a proto account with proto codec", memoryNodeFn: NewInMemoryTendermintNodeProto, upgrades: &upgrades{codecUpgrade{true, 2}}},
	}
	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			if tc.upgrades != nil { // NOTE: Use to perform neccesary upgrades for test
				codec.UpgradeHeight = tc.upgrades.codecUpgrade.height
				_ = memCodecMod(tc.upgrades.codecUpgrade.upgradeMod)
			}
			_, kb, cleanup := tc.memoryNodeFn(t, oneValTwoNodeGenesisState())
			time.Sleep(2 * time.Second)
			cb, err := kb.GetCoinbase()
			assert.Nil(t, err)
			_, _, evtChan := subscribeTo(t, tmTypes.EventNewBlock)
			var tx *sdk.TxResponse
			<-evtChan // Wait for block
			memCli, stopCli, evtChan := subscribeTo(t, tmTypes.EventTx)
			tx, err = gov.DAOTransferTx(memCodec(), memCli, kb, cb.GetAddress(), nil, sdk.OneInt(), govTypes.DAOBurn.String(), "test", 1000000, tc.codecUpgrade.upgradeMod)
			assert.Nil(t, err)
			assert.NotNil(t, tx)

			<-evtChan // Wait for tx
			balance, err := PCA.QueryDaoBalance(PCA.LastBlockHeight())
			assert.Nil(t, err)
			assert.True(t, balance.Equal(sdk.NewInt(999)))

			cleanup()
			stopCli()
		})
	}
}

func TestClaimAminoTx(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping in short mode")
	}
	tt := []struct {
		name         string
		memoryNodeFn func(t *testing.T, genesisState []byte) (tendermint *node.Node, keybase keys.Keybase, cleanup func())
		*upgrades
	}{
		{name: "claim tx from amino with amino codec ", memoryNodeFn: NewInMemoryTendermintNodeAmino, upgrades: &upgrades{codecUpgrade{false, 7000}}},
		//{name: "claim tx from a proto with proto codec", memoryNodeFn: NewInMemoryTendermintNodeProto, upgrades: &upgrades{codecUpgrade{true, 4}}},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			if tc.upgrades != nil { // NOTE: Use to perform neccesary upgrades for test
				codec.UpgradeHeight = tc.upgrades.codecUpgrade.height
				_ = memCodecMod(tc.upgrades.codecUpgrade.upgradeMod)
			}
			genBz, _, validators, app := fiveValidatorsOneAppGenesis()
			kb := getInMemoryKeybase()
			_, _, cleanup := tc.memoryNodeFn(t, genBz)
			time.Sleep(2 * time.Second)
			for i := 0; i < 5; i++ {
				appPrivateKey, err := kb.ExportPrivateKeyObject(app.Address, "test")
				assert.Nil(t, err)
				// setup AAT
				aat := pocketTypes.AAT{
					Version:              "0.0.1",
					ApplicationPublicKey: appPrivateKey.PublicKey().RawString(),
					ClientPublicKey:      appPrivateKey.PublicKey().RawString(),
					ApplicationSignature: "",
				}
				sig, err := appPrivateKey.Sign(aat.Hash())
				if err != nil {
					panic(err)
				}
				aat.ApplicationSignature = hex.EncodeToString(sig)
				proof := pocketTypes.RelayProof{
					Entropy:            int64(rand.Int()),
					RequestHash:        hex.EncodeToString(pocketTypes.Hash([]byte("fake"))),
					SessionBlockHeight: 1,
					ServicerPubKey:     validators[0].PublicKey.RawString(),
					Blockchain:         sdk.PlaceholderHash,
					Token:              aat,
					Signature:          "",
				}
				sig, err = appPrivateKey.Sign(proof.Hash())
				if err != nil {
					t.Fatal(err)
				}
				proof.Signature = hex.EncodeToString(sig)
				pocketTypes.SetProof(pocketTypes.SessionHeader{
					ApplicationPubKey:  appPrivateKey.PublicKey().RawString(),
					Chain:              sdk.PlaceholderHash,
					SessionBlockHeight: 1,
				}, pocketTypes.RelayEvidence, proof, sdk.NewInt(1000000))
				assert.Nil(t, err)
			}
			_, _, evtChan := subscribeTo(t, tmTypes.EventTx)
			res := <-evtChan
			fmt.Println(res)
			if res.Events["message.action"][0] != pocketTypes.EventTypeClaim {
				t.Fatal("claim message was not received first")
			}
			_, stopCli, evtChan := subscribeTo(t, tmTypes.EventTx)
			res = <-evtChan
			if res.Events["message.action"][0] != pocketTypes.EventTypeProof {
				t.Fatal("proof message was not received afterward")
			}
			cleanup()
			stopCli()
		})
	}
}

func TestClaimProtoTx(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping in short mode")
	}
	tt := []struct {
		name         string
		memoryNodeFn func(t *testing.T, genesisState []byte) (tendermint *node.Node, keybase keys.Keybase, cleanup func())
		*upgrades
	}{
		//{name: "claim tx from amino with amino codec ", memoryNodeFn: NewInMemoryTendermintNodeAmino, upgrades: &upgrades{codecUpgrade{false, 7000}}},
		{name: "claim tx from a proto with proto codec", memoryNodeFn: NewInMemoryTendermintNodeProto, upgrades: &upgrades{codecUpgrade{true, 5}}},
	}
	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			if tc.upgrades != nil { // NOTE: Use to perform neccesary upgrades for test
				codec.UpgradeHeight = tc.upgrades.codecUpgrade.height
				_ = memCodecMod(tc.upgrades.codecUpgrade.upgradeMod)
			}
			genBz, _, validators, app := fiveValidatorsOneAppGenesis()
			kb := getInMemoryKeybase()
			_, _, cleanup := tc.memoryNodeFn(t, genBz)
			time.Sleep(2 * time.Second)
			for i := 0; i < 5; i++ {
				appPrivateKey, err := kb.ExportPrivateKeyObject(app.Address, "test")
				assert.Nil(t, err)
				// setup AAT
				aat := pocketTypes.AAT{
					Version:              "0.0.1",
					ApplicationPublicKey: appPrivateKey.PublicKey().RawString(),
					ClientPublicKey:      appPrivateKey.PublicKey().RawString(),
					ApplicationSignature: "",
				}
				sig, err := appPrivateKey.Sign(aat.Hash())
				if err != nil {
					panic(err)
				}
				aat.ApplicationSignature = hex.EncodeToString(sig)
				proof := pocketTypes.RelayProof{
					Entropy:            int64(rand.Int()),
					RequestHash:        hex.EncodeToString(pocketTypes.Hash([]byte("fake"))),
					SessionBlockHeight: 1,
					ServicerPubKey:     validators[0].PublicKey.RawString(),
					Blockchain:         sdk.PlaceholderHash,
					Token:              aat,
					Signature:          "",
				}
				sig, err = appPrivateKey.Sign(proof.Hash())
				if err != nil {
					t.Fatal(err)
				}
				proof.Signature = hex.EncodeToString(sig)
				pocketTypes.SetProof(pocketTypes.SessionHeader{
					ApplicationPubKey:  appPrivateKey.PublicKey().RawString(),
					Chain:              sdk.PlaceholderHash,
					SessionBlockHeight: 1,
				}, pocketTypes.RelayEvidence, proof, sdk.NewInt(1000000))
				assert.Nil(t, err)
			}
			_, _, evtChan := subscribeTo(t, tmTypes.EventTx)
			res := <-evtChan
			fmt.Println(res)
			if res.Events["message.action"][0] != pocketTypes.EventTypeClaim {
				t.Fatal("claim message was not received first")
			}
			_, stopCli, evtChan := subscribeTo(t, tmTypes.EventTx)
			res = <-evtChan
			if res.Events["message.action"][0] != pocketTypes.EventTypeProof {
				t.Fatal("proof message was not received afterward")
			}
			cleanup()
			stopCli()
		})
	}
}

func TestAminoClaimTxChallenge(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping in short mode")
	}
	tt := []struct {
		name         string
		memoryNodeFn func(t *testing.T, genesisState []byte) (tendermint *node.Node, keybase keys.Keybase, cleanup func())
		*upgrades
	}{
		{name: "challenge a claim tx from amino with amino codec ", memoryNodeFn: NewInMemoryTendermintNodeAmino, upgrades: &upgrades{codecUpgrade{false, 7000}}},
		//{name: "challenge a claim tx from a proto with proto codec", memoryNodeFn: NewInMemoryTendermintNodeProto, upgrades: &upgrades{codecUpgrade{true, 2}}}, // TODO: FULL PROT SCENARIO
	}
	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			genBz, keys, _, _ := fiveValidatorsOneAppGenesis()
			challenges := NewValidChallengeProof(t, keys, 5)
			_, _, cleanup := tc.memoryNodeFn(t, genBz)
			for _, c := range challenges {
				c.Store(sdk.NewInt(1000000))
			}
			_, _, evtChan := subscribeTo(t, tmTypes.EventTx)
			res := <-evtChan // Wait for tx
			if res.Events["message.action"][0] != pocketTypes.EventTypeClaim {
				t.Fatal("claim message was not received first")
			}

			_, stopCli, evtChan := subscribeTo(t, tmTypes.EventTx)
			res = <-evtChan // Wait for tx
			if res.Events["message.action"][0] != pocketTypes.EventTypeProof {
				t.Fatal("proof message was not received afterward")
			}
			cleanup()
			stopCli()
		})
	}
}

func TestProtoClaimTxChallenge(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping in short mode")
	}
	tt := []struct {
		name         string
		memoryNodeFn func(t *testing.T, genesisState []byte) (tendermint *node.Node, keybase keys.Keybase, cleanup func())
		*upgrades
	}{
		//{name: "challenge a claim tx from amino with amino codec ", memoryNodeFn: NewInMemoryTendermintNodeAmino, upgrades: &upgrades{codecUpgrade{false, 7000}}},
		{name: "challenge a claim tx from a proto with proto codec", memoryNodeFn: NewInMemoryTendermintNodeProto, upgrades: &upgrades{codecUpgrade{true, 2}}}, // TODO: FULL PROT SCENARIO
	}
	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			genBz, keys, _, _ := fiveValidatorsOneAppGenesis()
			challenges := NewValidChallengeProof(t, keys, 5)
			_, _, cleanup := tc.memoryNodeFn(t, genBz)
			for _, c := range challenges {
				c.Store(sdk.NewInt(1000000))
			}
			_, _, evtChan := subscribeTo(t, tmTypes.EventTx)
			res := <-evtChan // Wait for tx
			if res.Events["message.action"][0] != pocketTypes.EventTypeClaim {
				t.Fatal("claim message was not received first")
			}

			_, stopCli, evtChan := subscribeTo(t, tmTypes.EventTx)
			res = <-evtChan // Wait for tx
			if res.Events["message.action"][0] != pocketTypes.EventTypeProof {
				t.Fatal("proof message was not received afterward")
			}
			cleanup()
			stopCli()
		})
	}
}

func NewValidChallengeProof(t *testing.T, privateKeys []crypto.PrivateKey, numOfChallenges int) (challenge []pocketTypes.ChallengeProofInvalidData) {

	appPrivateKey := privateKeys[1]
	servicerPrivKey1 := privateKeys[4]
	servicerPrivKey2 := privateKeys[2]
	servicerPrivKey3 := privateKeys[3]
	clientPrivateKey := servicerPrivKey3
	appPubKey := appPrivateKey.PublicKey().RawString()
	servicerPubKey := servicerPrivKey1.PublicKey().RawString()
	servicerPubKey2 := servicerPrivKey2.PublicKey().RawString()
	servicerPubKey3 := servicerPrivKey3.PublicKey().RawString()
	reporterPrivKey := privateKeys[0]
	reporterPubKey := reporterPrivKey.PublicKey()
	reporterAddr := reporterPubKey.Address()
	clientPubKey := clientPrivateKey.PublicKey().RawString()
	var proofs []pocketTypes.ChallengeProofInvalidData
	for i := 0; i < numOfChallenges; i++ {
		validProof := pocketTypes.RelayProof{
			Entropy:            int64(rand.Intn(500000)),
			SessionBlockHeight: 1,
			ServicerPubKey:     servicerPubKey,
			RequestHash:        clientPubKey, // fake
			Blockchain:         sdk.PlaceholderHash,
			Token: pocketTypes.AAT{
				Version:              "0.0.1",
				ApplicationPublicKey: appPubKey,
				ClientPublicKey:      clientPubKey,
				ApplicationSignature: "",
			},
			Signature: "",
		}
		appSignature, er := appPrivateKey.Sign(validProof.Token.Hash())
		if er != nil {
			t.Fatalf(er.Error())
		}
		validProof.Token.ApplicationSignature = hex.EncodeToString(appSignature)
		clientSignature, er := clientPrivateKey.Sign(validProof.Hash())
		if er != nil {
			t.Fatalf(er.Error())
		}
		validProof.Signature = hex.EncodeToString(clientSignature)
		// valid proof 2
		validProof2 := pocketTypes.RelayProof{
			Entropy:            0,
			SessionBlockHeight: 1,
			ServicerPubKey:     servicerPubKey2,
			RequestHash:        clientPubKey, // fake
			Blockchain:         sdk.PlaceholderHash,
			Token: pocketTypes.AAT{
				Version:              "0.0.1",
				ApplicationPublicKey: appPubKey,
				ClientPublicKey:      clientPubKey,
				ApplicationSignature: "",
			},
			Signature: "",
		}
		appSignature, er = appPrivateKey.Sign(validProof2.Token.Hash())
		if er != nil {
			t.Fatalf(er.Error())
		}
		validProof2.Token.ApplicationSignature = hex.EncodeToString(appSignature)
		clientSignature, er = clientPrivateKey.Sign(validProof2.Hash())
		if er != nil {
			t.Fatalf(er.Error())
		}
		validProof2.Signature = hex.EncodeToString(clientSignature)
		// valid proof 3
		validProof3 := pocketTypes.RelayProof{
			Entropy:            0,
			SessionBlockHeight: 1,
			ServicerPubKey:     servicerPubKey3,
			RequestHash:        clientPubKey, // fake
			Blockchain:         sdk.PlaceholderHash,
			Token: pocketTypes.AAT{
				Version:              "0.0.1",
				ApplicationPublicKey: appPubKey,
				ClientPublicKey:      clientPubKey,
				ApplicationSignature: "",
			},
			Signature: "",
		}
		appSignature, er = appPrivateKey.Sign(validProof3.Token.Hash())
		if er != nil {
			t.Fatalf(er.Error())
		}
		validProof3.Token.ApplicationSignature = hex.EncodeToString(appSignature)
		clientSignature, er = clientPrivateKey.Sign(validProof3.Hash())
		if er != nil {
			t.Fatalf(er.Error())
		}
		validProof3.Signature = hex.EncodeToString(clientSignature)
		// create responses
		majorityResponsePayload := `{"id":67,"jsonrpc":"2.0","result":"Mist/v0.9.3/darwin/go1.4.1"}`
		minorityResponsePayload := `{"id":67,"jsonrpc":"2.0","result":"Mist/v0.9.3/darwin/go1.4.2"}`
		// majority response 1
		majResp1 := pocketTypes.RelayResponse{
			Signature: "",
			Response:  majorityResponsePayload,
			Proof:     validProof,
		}
		sig, er := servicerPrivKey1.Sign(majResp1.Hash())
		if er != nil {
			t.Fatalf(er.Error())
		}
		majResp1.Signature = hex.EncodeToString(sig)
		// majority response 2
		majResp2 := pocketTypes.RelayResponse{
			Signature: "",
			Response:  majorityResponsePayload,
			Proof:     validProof2,
		}
		sig, er = servicerPrivKey2.Sign(majResp2.Hash())
		if er != nil {
			t.Fatalf(er.Error())
		}
		majResp2.Signature = hex.EncodeToString(sig)
		// minority response
		minResp := pocketTypes.RelayResponse{
			Signature: "",
			Response:  minorityResponsePayload,
			Proof:     validProof3,
		}
		sig, er = servicerPrivKey3.Sign(minResp.Hash())
		if er != nil {
			t.Fatalf(er.Error())
		}
		minResp.Signature = hex.EncodeToString(sig)
		// create valid challenge proof
		proofs = append(proofs, pocketTypes.ChallengeProofInvalidData{
			MajorityResponses: []pocketTypes.RelayResponse{
				majResp1,
				majResp2,
			},
			MinorityResponse: minResp,
			ReporterAddress:  sdk.Address(reporterAddr),
		})
	}
	return proofs
}
