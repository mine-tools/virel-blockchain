package main

import (
	"flag"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/virel-project/virel-blockchain/v3/address"
	"github.com/virel-project/virel-blockchain/v3/config"
	"github.com/virel-project/virel-blockchain/v3/logger"
	"github.com/virel-project/virel-blockchain/v3/transaction"
	"github.com/virel-project/virel-blockchain/v3/util"
	"github.com/virel-project/virel-blockchain/v3/util/updatechecker"
	"github.com/virel-project/virel-blockchain/v3/wallet"

	"github.com/ergochat/readline"
)

var Log = logger.New()

var default_rpc = fmt.Sprintf("http://127.0.0.1:%d", config.RPC_BIND_PORT)

func initialPrompt(daemon_address string) *wallet.Wallet {
	l, err := readline.NewEx(&readline.Config{
		InterruptPrompt: "^C",
		EOFPrompt:       "exit",

		HistorySearchFold: true,
	})
	if err != nil {
		panic(err)
	}
	defer l.Close()

	Log.SetStdout(l.Stdout())
	Log.SetStderr(l.Stderr())

	Log.Info("Starting", config.NETWORK_NAME, "CLI wallet")
	if config.NETWORK_NAME != "mainnet" {
		Log.Warn("This is a", strings.ToUpper(config.NETWORK_NAME), "node, only for testing the blockchain.")
		Log.Warn("Be aware that any amount transacted in", config.NETWORK_NAME, "is worthless.")
	}

	lcfg := l.GeneratePasswordConfig()
	lcfg.MaskRune = '*'

	for {
		Log.Info("Available commands:")
		Log.Info("open    Open wallet file")
		Log.Info("create  Creates a new wallet")
		Log.Info("restore Restore a wallet from seedphrase")

		l.SetPrompt("\033[32m>\033[0m ")

		line, err := l.ReadLine()
		if err != nil {
			Log.Err(err)
			os.Exit(0)
		}

		cmds := strings.Split(strings.ToLower(strings.ReplaceAll(line, "  ", " ")), " ")

		if len(cmds) == 0 {
			Log.Err("invalid command")
			continue
		}

		cmd := cmds[0]
		if cmd != "open" && cmd != "create" && cmd != "restore" {
			Log.Err("unknown command")
			continue
		}
		if len(cmds) == 1 {
			l.SetPrompt("Wallet name: ")
			filename, err := l.ReadLine()
			if err != nil {
				Log.Err(err)
				os.Exit(0)
			}
			l.SetPrompt("\033[32m>\033[0m ")

			if len(filename) == 0 {
				Log.Err("wallet name is too short")
				continue
			}

			cmds = append(cmds, filename)
		}

		fmt.Print("Wallet password: ")
		password, err := l.ReadLineWithConfig(lcfg)
		if err != nil {
			Log.Err(err)
		}

		if cmds[0] == "open" {
			Log.Info("opening wallet")

			w, err := wallet.OpenWalletFile(daemon_address, cmds[1]+".keys", password)
			if err != nil {
				Log.Err(err)
				continue
			}
			return w
		} else {
			fmt.Print("Repeat password: ")
			confirmPass, err := l.ReadLineWithConfig(lcfg)
			if err != nil {
				Log.Err(err)
				continue
			}
			if string(confirmPass) != string(password) {
				Log.Err("password doesn't match")
				continue
			}

			if cmd == "create" {
				w, err := wallet.CreateWalletFile(daemon_address, cmds[1]+".keys", password)
				if err != nil {
					Log.Err("Could not create wallet:", err)
					continue
				}

				return w
			} else if cmd == "restore" {
				Log.Info("restoring wallet")

				l.SetPrompt("Mnemonic seed: ")
				mnemonic, err := l.ReadLine()
				if err != nil {
					Log.Err(err)
					os.Exit(0)
				}

				w, err := wallet.CreateWalletFileFromMnemonic(daemon_address, cmds[1]+".keys",
					mnemonic, password)
				if err != nil {
					Log.Err(err)
					continue
				}
				return w
			}
		}
	}
}

func main() {
	version := flag.Bool("version", false, "prints version and exits")
	log_level := flag.Uint("log-level", 1, "sets the log level (range: 0-3)")
	rpc_bind_ip := flag.String("rpc-bind-ip", "0.0.0.0", "starts RPC server on this IP")
	rpc_bind_port := flag.Uint("rpc-bind-port", 0, "starts RPC server on this port")
	rpc_auth := flag.String("rpc-auth", "", "colon-separated username and password, like user:pass")
	open_wallet := flag.String("open-wallet", "", "open a wallet file")
	wallet_password := flag.String("wallet-password", "", "wallet password when using --open-wallet")
	non_interactive := flag.Bool("non-interactive", false, "if set, the node will not process the stdinput. Useful for running as a service.")
	daemon_address := flag.String("daemon-address", default_rpc, "sets the daemon")
	start_staking := flag.String("start-staking", "", "starts staking to the provided delegate id")
	mnemonic := flag.String("mnemonic", "", "mnemonic seed phrase for wallet operations")
	transfer_dest := flag.String("transfer-dest", "", "destination address for transfer")
	transfer_amount := flag.String("transfer-amount", "", "amount to transfer (in coins)")
	generate_wallet := flag.Bool("generate", false, "generate a new wallet and save to file")
	generate_wallet_name := flag.String("generate-name", "", "wallet name for --generate (default: auto-generated from timestamp)")
	generate_password := flag.String("generate-password", "", "password for generated wallet (if empty, wallet will be unencrypted)")

	flag.Parse()

	if *version {
		fmt.Printf("%s-wallet-cli v%v.%v.%v", config.NAME, config.VERSION_MAJOR, config.VERSION_MINOR, config.VERSION_PATCH)
		os.Exit(0)
	}

	Log.SetLogLevel(uint8(*log_level))

	Log.Infof("Starting Virel Wallet CLI v%v.%v.%v", config.VERSION_MAJOR, config.VERSION_MINOR, config.VERSION_PATCH)

	go updatechecker.RunUpdateChecker(Log, config.UPDATE_CHECK_URL, config.VERSION_MAJOR, config.VERSION_MINOR, config.VERSION_PATCH)

	var w *wallet.Wallet

	// 检查是否要生成新钱包
	if *generate_wallet {
		err := generateNewWallet(*daemon_address, *generate_wallet_name, *generate_password)
		if err != nil {
			Log.Fatal(err)
		}
		os.Exit(0)
	}

	// 检查是否要从助记词查询余额
	if len(*mnemonic) > 0 && len(*transfer_dest) == 0 && len(*transfer_amount) == 0 {
		err := queryBalanceFromMnemonic(*daemon_address, *mnemonic)
		if err != nil {
			Log.Fatal(err)
		}
		os.Exit(0)
	}

	// 检查是否要从助记词直接转账
	if len(*mnemonic) > 0 && len(*transfer_dest) > 0 && len(*transfer_amount) > 0 {
		err := transferFromMnemonic(*daemon_address, *mnemonic, *transfer_dest, *transfer_amount)
		if err != nil {
			Log.Fatal(err)
		}
		os.Exit(0)
	}

	if len(*open_wallet) > 0 {
		wallname := *open_wallet

		if strings.ContainsAny(wallname, "/. \\$") {
			Log.Fatal("invalid wallet name")
		}
		var err error
		w, err = wallet.OpenWalletFile(*daemon_address, wallname+".keys", *wallet_password)
		if err != nil {
			Log.Fatal(err)
		}
	} else {
		w = initialPrompt(*daemon_address)
	}

	if *rpc_bind_port != 0 {
		if len(*rpc_auth) < 7 {
			Log.Err("rpc-auth is invalid or too short")
		}
		Log.Infof("Starting wallet rpc server on %v:%v", *rpc_bind_ip, *rpc_bind_port)
		startRpcServer(w, *rpc_bind_ip, uint16(*rpc_bind_port), *rpc_auth)
	}

	addr := w.GetAddress()
	if addr.Addr == address.INVALID_ADDRESS {
		Log.Fatal("wallet has invalid address")
	}

	Log.Info("Wallet", addr, "has been loaded")
	Log.Infof("Public key: %x", w.GetPubKey())

	Log.Debugf("Address hex: %x", addr.Addr[:])

	if len(*start_staking) > 0 {
		delegateAddr, err := address.FromString(*start_staking)
		if err != nil {
			Log.Err("invalid delegate address:", err)
			return
		}

		go staker(w, delegateAddr.Addr)
	}

	err := w.Refresh()
	if err != nil {
		Log.Warn("refresh failed:", err)
	} else {
		Log.Info("Balance:", util.FormatCoin(w.GetBalance()))
		Log.Infof("Last nonce: %d", w.GetLastNonce())
		if w.GetDelegateId() != 0 {
			Log.Infof("Delegate: %s (%s)", w.GetDelegateName(), address.NewDelegateAddress(w.GetDelegateId()))
			Log.Infof("Staked balance: %s", util.FormatCoin(w.GetStakedBalance()))
		}
	}

	if !*non_interactive {
		prompts(w)
	} else {
		// wait forever
		c := make(chan bool)
		Log.Err(<-c)
	}
}

// transferFromMnemonic 从助记词直接创建钱包并执行转账
func transferFromMnemonic(daemonAddress, mnemonic, destStr, amountStr string) error {
	Log.Info("Creating wallet from mnemonic for transfer...")

	// 从助记词创建临时钱包（不保存文件）
	w, _, err := wallet.CreateWalletFromMnemonic(daemonAddress, mnemonic, "", false)
	if err != nil {
		return fmt.Errorf("failed to create wallet from mnemonic: %w", err)
	}

	Log.Info("Wallet created from mnemonic")
	Log.Infof("Wallet address: %s", w.GetAddress())

	// 刷新钱包状态以获取余额和nonce
	err = w.Refresh()
	if err != nil {
		return fmt.Errorf("failed to refresh wallet: %w", err)
	}

	Log.Infof("Balance: %s", util.FormatCoin(w.GetBalance()))
	Log.Infof("Last nonce: %d", w.GetLastNonce())

	// 解析目标地址
	dst, err := address.FromString(destStr)
	if err != nil {
		return fmt.Errorf("invalid destination address: %w", err)
	}

	// 解析转账金额
	amtFloat, err := strconv.ParseFloat(amountStr, 64)
	if err != nil || amtFloat <= 0 {
		return fmt.Errorf("invalid amount: %s", amountStr)
	}

	amt := uint64(amtFloat * config.COIN)
	if amt < 1 {
		return fmt.Errorf("amount too small")
	}

	// 检查余额
	if amt > w.GetBalance() {
		return fmt.Errorf("insufficient balance: have %s, need %s", util.FormatCoin(w.GetBalance()), util.FormatCoin(amt))
	}

	// 创建转账交易
	outputs := []transaction.Output{
		{
			Amount:    amt,
			Recipient: dst.Addr,
			PaymentId: dst.PaymentId,
		},
	}

	hasVersion := w.GetHeight() >= config.HARDFORK_V2_HEIGHT
	txn, err := w.Transfer(outputs, hasVersion)
	if err != nil {
		return fmt.Errorf("failed to create transfer transaction: %w", err)
	}

	Log.Infof("Transaction created:")
	Log.Infof("  Destination: %s", dst)
	Log.Infof("  Amount: %s", util.FormatCoin(amt))
	Log.Infof("  Fee: %s", util.FormatCoin(txn.Fee))

	// 提交交易
	submitRes, err := w.SubmitTx(txn)
	if err != nil {
		return fmt.Errorf("failed to submit transaction: %w", err)
	}

	Log.Infof("Transaction submitted successfully!")
	Log.Infof("Transaction ID: %s", submitRes.TXID.String())

	return nil
}

// queryBalanceFromMnemonic 从助记词创建临时钱包并查询余额
func queryBalanceFromMnemonic(daemonAddress, mnemonic string) error {
	Log.Info("Creating wallet from mnemonic to query balance...")

	// 从助记词创建临时钱包（不保存文件）
	w, _, err := wallet.CreateWalletFromMnemonic(daemonAddress, mnemonic, "", false)
	if err != nil {
		return fmt.Errorf("failed to create wallet from mnemonic: %w", err)
	}

	Log.Info("Wallet created from mnemonic")
	Log.Infof("Wallet address: %s", w.GetAddress())
	Log.Infof("Public key: %x", w.GetPubKey())

	// 刷新钱包状态以获取余额和nonce
	err = w.Refresh()
	if err != nil {
		return fmt.Errorf("failed to refresh wallet: %w", err)
	}

	// 显示钱包信息
	Log.Info("=" + strings.Repeat("=", 60))
	Log.Infof("Wallet Information:")
	Log.Infof("  Address: %s", w.GetAddress())
	Log.Infof("  Balance: %s", util.FormatCoin(w.GetBalance()))
	Log.Infof("  Last nonce: %d", w.GetLastNonce())
	Log.Infof("  Mempool balance: %s", util.FormatCoin(w.GetMempoolBalance()))
	Log.Infof("  Mempool nonce: %d", w.GetMempoolLastNonce())

	if w.GetDelegateId() != 0 {
		Log.Infof("  Delegate: %s (%s)", w.GetDelegateName(), address.NewDelegateAddress(w.GetDelegateId()))
		Log.Infof("  Staked balance: %s", util.FormatCoin(w.GetStakedBalance()))
	} else {
		Log.Info("  Delegate: Not set")
	}

	Log.Info("=" + strings.Repeat("=", 60))

	return nil
}

// generateNewWallet 生成新钱包并保存到文件（包含助记词和地址信息）
func generateNewWallet(daemonAddress, walletName, password string) error {
	Log.Info("Generating new wallet...")

	// 如果没有指定钱包名称，自动生成一个（基于时间戳）
	if walletName == "" {
		walletName = fmt.Sprintf("wallet_%d", time.Now().Unix())
	}

	// 检查钱包名称是否有效
	if strings.ContainsAny(walletName, "/. \\$") {
		return fmt.Errorf("invalid wallet name: contains invalid characters")
	}

	walletFile := walletName + ".keys"
	infoFile := "wallets.txt" // 统一的信息文件，每次追加

	// 检查钱包文件是否已存在
	_, err := os.Lstat(walletFile)
	if err == nil {
		return fmt.Errorf("wallet file already exists: %s", walletFile)
	}

	// 创建钱包（在内存中，获取助记词）
	// 如果提供了密码，使用正常KDF；如果没有密码，使用空密码和快速KDF
	fastkdf := password == ""
	w, dbEnc, err := wallet.CreateWallet(daemonAddress, password, fastkdf)
	if err != nil {
		return fmt.Errorf("failed to generate wallet: %w", err)
	}

	// 保存加密的钱包文件
	err = os.WriteFile(walletFile, dbEnc, 0o600)
	if err != nil {
		return fmt.Errorf("failed to save wallet file: %w", err)
	}

	// 追加信息到统一的 wallets.txt 文件（每行一个：助记词 地址）
	infoLine := fmt.Sprintf("%s %s\n", w.GetMnemonic(), w.GetAddress())

	// 检查文件是否存在，如果不存在则创建（包含标题行）
	var infoContent []byte
	_, err = os.Lstat(infoFile)
	if err != nil {
		// 文件不存在，创建新文件并添加标题
		infoContent = []byte(fmt.Sprintf("# Virel Wallets - Generated: %s\n# Format: <mnemonic> <address>\n# ⚠️  WARNING: Keep this file secure! Never share your mnemonic!\n\n", time.Now().Format("2006-01-02 15:04:05")))
	}

	// 追加新的钱包信息行
	file, err := os.OpenFile(infoFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o600)
	if err != nil {
		os.Remove(walletFile)
		return fmt.Errorf("failed to open info file: %w", err)
	}
	defer file.Close()

	// 如果是新文件，先写入标题
	if len(infoContent) > 0 {
		_, err = file.Write(infoContent)
		if err != nil {
			os.Remove(walletFile)
			return fmt.Errorf("failed to write info file header: %w", err)
		}
	}

	// 追加钱包信息行
	_, err = file.WriteString(infoLine)
	if err != nil {
		os.Remove(walletFile)
		return fmt.Errorf("failed to append to info file: %w", err)
	}

	// 显示钱包信息
	Log.Info("=" + strings.Repeat("=", 70))
	Log.Info("🎉 New Wallet Generated Successfully!")
	Log.Info("=" + strings.Repeat("=", 70))
	Log.Info("")
	Log.Infof("📁 Wallet File: %s", walletFile)
	Log.Infof("📄 Info File: %s", infoFile)
	Log.Info("")
	Log.Infof("📝 Mnemonic Seed Phrase (%d words):", strings.Count(w.GetMnemonic(), " ")+1)
	Log.Infof("   %s", w.GetMnemonic())
	Log.Info("")
	Log.Infof("📍 Wallet Address: %s", w.GetAddress())
	Log.Infof("🔑 Public Key: %x", w.GetPubKey())
	Log.Info("")
	Log.Info("✅ Wallet saved successfully!")
	Log.Info("")
	Log.Info("⚠️  WARNING:")
	Log.Info("   - Never share your mnemonic seed phrase with anyone!")
	Log.Info("   - Anyone with your mnemonic can access your wallet!")
	Log.Info("   - Keep the info file (.txt) in a secure location!")
	Log.Info("")
	Log.Info("💡 Files created/updated:")
	Log.Infof("   - %s (encrypted wallet file)", walletFile)
	Log.Infof("   - %s (all wallets info, one line per wallet: mnemonic address)", infoFile)
	Log.Info("")
	Log.Info("=" + strings.Repeat("=", 70))

	return nil
}
