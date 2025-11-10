package daemonrpc

import (
	"github.com/virel-project/virel-blockchain/v3/address"
	"github.com/virel-project/virel-blockchain/v3/block"
	"github.com/virel-project/virel-blockchain/v3/chaintype"
	"github.com/virel-project/virel-blockchain/v3/transaction"
	"github.com/virel-project/virel-blockchain/v3/util"
	"github.com/virel-project/virel-blockchain/v3/util/enc"
)

type GetTransactionRequest struct {
	Txid util.Hash `json:"txid"`
}

type GetTransactionResponse struct {
	Signer      *address.Integrated       `json:"sender"`
	Inputs      []transaction.StateInput  `json:"inputs"`
	Outputs     []transaction.StateOutput `json:"outputs"`
	TotalAmount uint64                    `json:"total_amount"`
	Fee         uint64                    `json:"fee"`
	Nonce       uint64                    `json:"nonce"`
	Signature   enc.Hex                   `json:"signature"`
	Height      uint64                    `json:"height"`
	Coinbase    bool                      `json:"coinbase"`
	VirtualSize uint64                    `json:"virtual_size"`
}

type GetInfoRequest struct {
}
type GetInfoResponse struct {
	Height            uint64    `json:"height"`
	TopHash           util.Hash `json:"top_hash"`
	TotalSupply       uint64    `json:"total_supply"`
	CirculatingSupply uint64    `json:"circulating_supply"`
	MaxSupply         uint64    `json:"max_supply"`
	SupplyCap         uint64    `json:"supply_cap"`
	Burned            uint64    `json:"burned"`
	Stake             uint64    `json:"stake"`
	Coin              uint64    `json:"coin"`
	Difficulty        string    `json:"difficulty"`
	CumulativeDiff    string    `json:"cumulative_diff"`
	Target            int       `json:"target_block_time"`
	BlockReward       uint64    `json:"block_reward"`
	Version           string    `json:"version"`
	Connections       int       `json:"peers"`
}

type GetAddressRequest struct {
	Address string `json:"address"`
}
type GetAddressResponse struct {
	Balance         uint64 `json:"balance"`
	LastNonce       uint64 `json:"last_nonce"` // last nonce used
	LastIncoming    uint64 `json:"last_incoming"`
	MempoolBalance  uint64 `json:"mempool_balance"`    // unconfirmed balance, from mempool
	MempoolNonce    uint64 `json:"mempool_last_nonce"` // unconfirmed nonce, from mempool
	MempoolIncoming uint64 `json:"mempool_incoming"`
	DelegateId      uint64 `json:"delegate_id"`
	Height          uint64 `json:"height"`
}

type GetTxListRequest struct {
	Address      address.Integrated `json:"address"`
	TransferType string             `json:"transfer_type"` // incoming or outgoing
	Page         uint64             `json:"page"`
}
type GetTxListResponse struct {
	Transactions []util.Hash `json:"transactions"`
	MaxPage      uint64      `json:"max_page"`
}

type SubmitTransactionRequest struct {
	Hex enc.Hex `json:"hex"` // transaction data as hex string
}
type SubmitTransactionResponse struct {
	TXID util.Hash `json:"txid"`
}

type GetBlockByHashRequest struct {
	Hash util.Hash `json:"hash"`
}
type GetBlockByHeightRequest struct {
	Height uint64 `json:"height"`
}
type GetBlockResponse struct {
	Block            block.Block     `json:"block"`
	Hash             string          `json:"hash"`
	TotalReward      uint64          `json:"total_reward"`
	MinerReward      uint64          `json:"miner_reward"`
	StakerReward     uint64          `json:"staker_reward"`
	GovernanceReward uint64          `json:"governance_reward"`
	Miner            string          `json:"miner"`
	Delegate         address.Address `json:"delegate"`
	NextDelegate     address.Address `json:"next_delegate"`
}

type CalcPowRequest struct {
	Blob     enc.Hex   `json:"blob"`
	SeedHash util.Hash `json:"seed_hash"`
}
type CalcPowResponse struct {
	Hash util.Hash `json:"hash"`
}

type ValidateAddressRequest struct {
	Address string `json:"address"`
}
type ValidateAddressResponse struct {
	Address      string `json:"address"`
	Valid        bool   `json:"valid"`
	ErrorMessage string `json:"error_message,omitempty"`
	MainAddress  string `json:"main_address"`
	PaymentId    uint64 `json:"payment_id"`
}

type StateInfo struct {
	Address string           `json:"address"`
	State   *chaintype.State `json:"state"`
	Staked  uint64           `json:"staked"`
}

func (s *StateInfo) Total() uint64 {
	return s.State.Balance + s.Staked
}

type RichListRequest struct {
}
type RichListResponse struct {
	Richest []StateInfo
}

type SubmitStakeSignatureRequest struct {
	DelegateId uint64  `json:"delegate_id"` // the delegate who signed this hash
	Hash       enc.Hex `json:"hash"`        // the block hash
	Signature  enc.Hex `json:"signature"`   // the signature
}
type SubmitStakeSignatureResponse struct {
	Accepted     bool   `json:"accepted"`
	ErrorMessage string `json:"error_message"`
}

type GetDelegateRequest struct {
	DelegateId      uint64 `json:"delegate_id"`
	DelegateAddress string `json:"delegate_address"`
}
type GetDelegateResponse struct {
	Id          uint64                     `json:"id"`
	Address     address.Address            `json:"address"`
	Owner       address.Address            `json:"owner"`
	TotalAmount uint64                     `json:"total_amount"`
	Name        string                     `json:"name"`
	Funds       []*chaintype.DelegatedFund `json:"funds"`
}

type StakingWalletInfo struct {
	Address          address.Address `json:"address"`
	Amount           uint64          `json:"amount"`
	UnlockHeight     uint64          `json:"unlock_height"`
	RemainingBlocks  uint64          `json:"remaining_blocks"`  // 剩余区块数
	RemainingSeconds uint64          `json:"remaining_seconds"` // 剩余秒数
	DelegateId       uint64          `json:"delegate_id"`
	DelegateName     string          `json:"delegate_name"`
	DelegateAddress  address.Address `json:"delegate_address"`
}

type GetAllStakingWalletsRequest struct {
}

type GetAllStakingWalletsResponse struct {
	Height  uint64              `json:"height"`
	Wallets []StakingWalletInfo `json:"wallets"`
	Total   uint64              `json:"total"` // 总质押金额
	Count   uint64              `json:"count"` // 质押钱包总数
}

type WalletStakingInfo struct {
	Address address.Address `json:"address"` // 钱包地址
	Amount  uint64          `json:"amount"`  // 质押/取消质押金额
}

type DailyStakingStats struct {
	Date           string              `json:"date"`             // 日期 (YYYY-MM-DD)
	StakeAmount    uint64              `json:"stake_amount"`     // 新增质押量
	StakeCount     uint64              `json:"stake_count"`      // 质押交易数
	UnstakeAmount  uint64              `json:"unstake_amount"`   // 转出质押量
	UnstakeCount   uint64              `json:"unstake_count"`    // 取消质押交易数
	NetStakeAmount uint64              `json:"net_stake_amount"` // 净质押量
	StakeWallets   []WalletStakingInfo `json:"stake_wallets"`    // 质押的钱包列表
	UnstakeWallets []WalletStakingInfo `json:"unstake_wallets"`  // 取消质押的钱包列表
}

type GetDailyStakingStatsRequest struct {
	StartHeight uint64 `json:"start_height,omitempty"` // 起始区块高度（可选）
	EndHeight   uint64 `json:"end_height,omitempty"`   // 结束区块高度（可选）
	Days        uint64 `json:"days,omitempty"`         // 查询最近N天的数据（可选）
}

type GetDailyStakingStatsResponse struct {
	Height          uint64              `json:"height"`            // 当前区块高度
	StartHeight     uint64              `json:"start_height"`      // 实际查询的起始高度
	EndHeight       uint64              `json:"end_height"`        // 实际查询的结束高度
	DailyStats      []DailyStakingStats `json:"daily_stats"`       // 每日统计
	TotalStake      uint64              `json:"total_stake"`       // 总新增质押量
	TotalUnstake    uint64              `json:"total_unstake"`     // 总转出质押量
	TotalStakeTxs   uint64              `json:"total_stake_txs"`   // 质押交易总数
	TotalUnstakeTxs uint64              `json:"total_unstake_txs"` // 取消质押交易总数
}
