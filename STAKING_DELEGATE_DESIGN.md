# 质押奖励Delegate设计机制说明

## 当前设计机制

### 1. 随机选择Delegate的机制

质押奖励的delegate选择采用**基于质押权重的伪随机选择**机制：

#### 核心算法（`blockchain/staking.go:GetStaker`）

```go
func (bc *Blockchain) GetStaker(txn adb.Txn, hash util.Hash, stats *Stats) (*chaintype.Delegate, error) {
    // 1. 将区块hash转换为coin index（0 到 StakedAmount 之间的随机数）
    coinIndex := hashToCoinIndex(hash, stats.StakedAmount)
    
    // 2. 遍历所有delegate，按质押量累加
    var coinsSeen uint64
    err := bc.GetDelegates(txn, func(d *chaintype.Delegate) (bool, error) {
        coinsSeen += d.TotalAmount()
        // 3. 找到包含coinIndex的delegate
        if coinIndex <= coinsSeen {
            delegate = d
            return true, nil // 找到，停止遍历
        }
        return false, nil // 继续遍历
    })
    
    return delegate, nil
}
```

#### 工作原理

1. **随机数生成**：使用区块的 `PrevHash` 通过 `hashToCoinIndex` 函数生成一个 0 到总质押量之间的随机数
2. **权重分配**：每个delegate的质押量作为权重，质押量越大，被选中的概率越高
3. **确定性**：对于相同的区块hash，总是选择相同的delegate（保证网络一致性）

### 2. 挖矿时的Delegate确定流程

#### 在 `blockchain/mining.go:GetBlockTemplate` 中：

```go
// 1. 首先尝试从stake signature获取DelegateId
stakesig, err := bc.GetStakeSig(txn, bl.BlockStakedHash())
if err == nil {
    bl.DelegateId = stakesig.DelegateId
}

// 2. 如果没有stake signature，从上一个区块获取
if bl.DelegateId == 0 {
    stakedbl, err := bc.GetBlock(txn, bl.BlockStakedHash())
    if err == nil {
        bl.DelegateId = stakedbl.NextDelegateId
    }
}

// 3. 计算下一个区块的delegate（NextDelegateId）
nextdelegate, err := bc.GetStaker(txn, bl.PrevHash(), bc.GetStats(txn))
bl.NextDelegateId = nextdelegate.Id
```

#### 验证机制（`blockchain/blockchain.go:PrevalidateBlock`）

```go
// 验证当前区块的DelegateId必须匹配上一个区块的NextDelegateId
if oldblock.NextDelegateId != bl.DelegateId {
    return fmt.Errorf("block's delegate id %d doesn't match the old block's next delegate id %d", 
        bl.DelegateId, oldblock.NextDelegateId)
}
```

### 3. 奖励分配机制

#### PoS奖励分配（`blockchain/bc-stateadd.go:ApplyPosReward`）

1. **奖励比例**：
   - 区块奖励的40%分配给PoS（如果区块有有效的stake signature）
   - 如果没有stake signature，PoS奖励被销毁

2. **分配方式**：
   - 99%的奖励按质押比例分配给所有质押者
   - 1%作为手续费给delegate owner（从舍入误差中扣除）

3. **计算公式**：
   ```go
   addAmount = (fund.Amount * posReward * 99) / (totalStake * 100)
   ```

## 当前限制：矿工不能指定Delegate

### 原因

1. **验证限制**：区块验证时要求 `bl.DelegateId == oldblock.NextDelegateId`
2. **确定性要求**：NextDelegateId由上一个区块的hash确定，保证网络一致性
3. **防止操纵**：防止矿工通过选择特定delegate来操纵奖励分配

### 矿工可以控制的

- ✅ **Recipient地址**：矿工可以指定PoW奖励的接收地址（50%的区块奖励）
- ❌ **DelegateId**：不能指定，必须匹配上一个区块的NextDelegateId
- ❌ **NextDelegateId**：不能指定，必须通过GetStaker(bl.PrevHash())计算得出

### NextDelegateId的验证

在 `blockchain/blockchain.go:ApplyBlockToState` 中：

```go
if bl.Version > 0 {
    // 验证NextDelegateId是否正确
    nextstaker, err := bc.GetStaker(txn, bl.PrevHash(), stats)
    if err != nil {
        return fmt.Errorf("failed to get next staker: %w", err)
    }
    if nextstaker.Id != bl.NextDelegateId {
        return fmt.Errorf("block has invalid NextDelegateId %d, expected %d", 
            bl.NextDelegateId, nextstaker.Id)
    }
}
```

这意味着NextDelegateId必须严格匹配通过算法计算出的值，矿工不能随意指定。

## 如果要支持矿工指定NextDelegateId

### 当前限制

NextDelegateId在验证时会被严格检查，必须匹配通过 `GetStaker(bl.PrevHash())` 计算出的值。这确保了：
1. **确定性**：所有节点对同一个区块会计算出相同的NextDelegateId
2. **公平性**：防止矿工通过选择特定delegate来操纵奖励分配
3. **一致性**：保证网络对下一个区块的delegate选择达成共识

### 如果要允许矿工指定NextDelegateId

#### 方案1：完全允许（不推荐）

允许矿工指定任意有效的delegate作为NextDelegateId，但这会：
- 破坏随机性和公平性
- 可能导致网络分叉（不同节点接受不同的NextDelegateId）
- 需要修改共识机制

#### 方案2：允许从候选列表中选择（推荐）

允许矿工从多个有效的候选delegate中选择，但需要：
1. **定义候选规则**：例如，选择质押量前N的delegate作为候选
2. **验证机制**：确保指定的delegate在候选列表中
3. **共识机制**：所有节点必须使用相同的候选列表

#### 方案3：允许"建议"但不强制

保持当前验证逻辑，但允许矿工在挖矿时"建议"一个NextDelegateId：
- 如果建议的delegate恰好是系统计算的，使用它
- 否则，使用系统计算的
- 这样可以表达偏好，但不影响共识

### 实现示例（方案2）

```go
// blockchain/blockchain.go:ApplyBlockToState
// 修改验证逻辑
if bl.Version > 0 {
    nextstaker, err := bc.GetStaker(txn, bl.PrevHash(), stats)
    if err != nil {
        return fmt.Errorf("failed to get next staker: %w", err)
    }
    
    // 允许矿工指定，但必须在候选列表中
    if nextstaker.Id != bl.NextDelegateId {
        // 检查是否在候选列表中（例如：质押量前10的delegate）
        if !bc.isValidNextDelegateCandidate(txn, bl.NextDelegateId, stats) {
            return fmt.Errorf("block has invalid NextDelegateId %d, expected %d or valid candidate", 
                bl.NextDelegateId, nextstaker.Id)
        }
    }
}

func (bc *Blockchain) isValidNextDelegateCandidate(txn adb.Txn, delegateId uint64, stats *Stats) bool {
    // 获取所有delegate，按质押量排序
    // 检查delegateId是否在前N个候选delegate中
    // 例如：前10个质押量最大的delegate
    // ...
}
```

## 如果要支持矿工指定Delegate

### 方案1：允许矿工选择（需要修改验证逻辑）

#### 修改点1：区块验证逻辑

```go
// blockchain/blockchain.go:PrevalidateBlock
// 修改前：
if oldblock.NextDelegateId != bl.DelegateId {
    return fmt.Errorf("block's delegate id mismatch")
}

// 修改后：允许矿工指定，但必须验证delegate有效
if bl.DelegateId != oldblock.NextDelegateId {
    // 验证矿工指定的delegate是否有效
    delegate, err := bc.GetDelegate(tx, bl.DelegateId)
    if err != nil {
        return fmt.Errorf("invalid delegate id %d: %w", bl.DelegateId, err)
    }
    if delegate.TotalAmount() == 0 {
        return fmt.Errorf("delegate %d has no stake", bl.DelegateId)
    }
    // 可选：添加其他验证，如最小质押量要求
}
```

#### 修改点2：挖矿模板生成

```go
// blockchain/mining.go:GetBlockTemplate
// 添加参数允许指定delegate
func (bc *Blockchain) GetBlockTemplate(txn adb.Txn, addr address.Address, preferredDelegateId uint64) (*block.Block, uint64, error) {
    // ... 现有代码 ...
    
    // 如果指定了preferred delegate，使用它
    if preferredDelegateId > 0 {
        delegate, err := bc.GetDelegate(txn, preferredDelegateId)
        if err == nil && delegate.TotalAmount() > 0 {
            bl.DelegateId = preferredDelegateId
        }
    } else {
        // 使用原有逻辑
        // ...
    }
}
```

#### 修改点3：挖矿接口

```go
// 在StartMining或GetBlockTemplate中添加delegate参数
func (bc *Blockchain) StartMining(addr address.Address, preferredDelegateId uint64) {
    // ...
}
```

### 方案2：保持当前机制，但添加可选参数

保持验证逻辑不变，但允许矿工在挖矿时"建议"一个delegate，如果该delegate恰好是系统选择的，则使用它。这样可以：
- 保持网络一致性
- 允许矿工表达偏好（如果他们的偏好与系统选择一致）

### 方案3：混合机制

- 如果矿工指定的delegate有效且有质押，使用矿工指定的
- 否则，使用系统自动选择的
- 这样可以鼓励delegate提供更好的服务来吸引矿工

## 实现建议

如果要实现矿工指定delegate功能，建议：

1. **添加配置选项**：允许节点选择是否启用此功能
2. **验证机制**：确保指定的delegate有效且有足够的质押
3. **向后兼容**：如果不指定，使用原有逻辑
4. **文档说明**：清楚说明此功能的影响和风险

## 当前代码位置

- **Delegate选择逻辑**：`blockchain/staking.go:GetStaker`
- **挖矿模板生成**：`blockchain/mining.go:GetBlockTemplate`
- **区块验证**：`blockchain/blockchain.go:PrevalidateBlock`
- **奖励分配**：`blockchain/bc-stateadd.go:ApplyPosReward`
- **Coinbase奖励**：`block/coinbase.go:CoinbaseTransaction`

