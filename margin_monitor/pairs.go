package margin_monitor

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/redis/go-redis/v9"
	"log"
	"margin_monitor/config"
	"os"
	"sort"
	"strings"
	"time"
)

type Pair struct {
	Conf        *config.Config
	RedisClient *redis.Client
}

func NewPair(conf *config.Config) *Pair {
	rdb := redis.NewClient(&redis.Options{
		Addr:         conf.RefreshPairs.Redis.URL,
		Password:     conf.RefreshPairs.Redis.Password,
		DB:           conf.RefreshPairs.Redis.DB,
		PoolSize:     conf.RefreshPairs.Redis.PoolSize,
		MinIdleConns: conf.RefreshPairs.Redis.MinIdleConns,
		MaxRetries:   conf.RefreshPairs.Redis.MaxRetries,
	})
	if err := rdb.Ping(context.Background()).Err(); err != nil {
		log.Fatalf("failed to connect to redis: %v", err)
	}
	return &Pair{
		Conf:        conf,
		RedisClient: rdb,
	}
}

func (p *Pair) CollectPair(name string) map[string]BacktestResult {
	ctx := context.Background()
	prefix := name + ":"
	result := make(map[string]BacktestResult)
	cursor := uint64(0)

	start := time.Now()

	for {
		keys, newCursor, err := p.RedisClient.Scan(ctx, cursor, prefix+"*", 100).Result()
		if err != nil {
			log.Printf("⚠️ Error scanning Redis keys: %v", err)
			break
		}
		cursor = newCursor

		if len(keys) > 0 {
			values, err := p.RedisClient.MGet(ctx, keys...).Result()
			if err != nil {
				log.Printf("⚠️ Error getting Redis values: %v", err)
				continue
			}

			for i, key := range keys {
				val := values[i]
				if val == nil {
					continue
				}

				btResult, err := ParseBacktestValue(fmt.Sprintf("%v", val))
				if err != nil {
					log.Printf("⚠️ Failed to parse value for key %s: %v", key, err)
					continue
				}
				result[key] = btResult
			}
		}

		if cursor == 0 {
			break
		}
	}

	log.Printf("✅ Collected %d structured backtest results with prefix '%s' in %s",
		len(result), prefix, time.Since(start))
	return result
}

type BacktestResult struct {
	Success int
	Fail    int
	Ratio   float64
}

func ParseBacktestValue(value string) (BacktestResult, error) {
	var res BacktestResult
	_, err := fmt.Sscanf(value, "%d %d %f", &res.Success, &res.Fail, &res.Ratio)
	return res, err
}

type pairEntry struct {
	Key    string
	Result BacktestResult
}

// checkPairs 检查和更新交易对
func (c *Controller) checkPairs() {
	for i := range c.Conf.RefreshPairs.Bot {
		bot := c.Conf.RefreshPairs.Bot[i]
		go c.processBot(bot)
	}
}

// processBot 处理单个 bot 的交易对更新逻辑
func (c *Controller) processBot(bot config.Bot) {
	pairMap := c.Pair.CollectPair(bot.Name)

	// 构造 pairList
	var pairList []pairEntry
	for k, v := range pairMap {
		pairList = append(pairList, pairEntry{
			Key:    formatPairKey(k),
			Result: v,
		})
	}

	// 排序
	sort.Slice(pairList, func(i, j int) bool {
		return pairList[i].Result.Ratio > pairList[j].Result.Ratio
	})

	// 写入文件
	writeJSONAsyncWithTimestamp(bot.Name, pairList)

	// 过滤掉收益 < 0 的
	filtered := make([]pairEntry, 0, len(pairList))
	for _, p := range pairList {
		if p.Result.Ratio >= 0 {
			filtered = append(filtered, p)
		}
	}
	pairList = filtered

	// 处理 top config
	topPair, err := c.Pair.HandleTopPair(pairList, bot)
	if err != nil {
		log.Printf("Failed to update top config for bot %s: %v", bot.Name, err)
		return
	}
	if len(topPair) > 0 {
		NotifyPairUpdate(topPair, fmt.Sprintf("%s: top pairs updated successfully:", bot.Name), c.M.SendTelegramMessage, 40)
	} else {
		log.Printf("topPair 为空")
	}
	log.Printf("%s top 币处理完成", bot.Name)
	log.Println("等待防止触发 API 限速, 等待10min 处理下一个机器人")
	// 等待防止触发 API 限速
	time.Sleep(10 * time.Minute)

	// 处理次级配置
	seekPairs, err := c.Pair.HandlePair(pairList, bot)
	if err != nil {
		log.Printf("Failed to update pair config for bot %s: %v", bot.Name, err)
		return
	}
	if len(seekPairs) > 0 {
		NotifyPairUpdate(seekPairs, fmt.Sprintf("%s: pairs updated successfully:", bot.Name), c.M.SendTelegramMessage, 40)
	} else {
		log.Printf("seekPairs 为空")
	}
}

func formatPairKey(raw string) string {
	// 去掉前缀（如 *:backtest:）
	parts := strings.Split(raw, ":")
	if len(parts) < 3 {
		return raw
	}
	symbolParts := strings.Split(parts[2], "_")
	if len(symbolParts) != 3 {
		return raw
	}
	return fmt.Sprintf("%s/%s:%s", symbolParts[0], symbolParts[1], symbolParts[2])
}

func NotifyPairUpdate(pairs []pairEntry, title string, sendFunc func(string), chunkSize int) {
	if len(pairs) == 0 {
		sendFunc(fmt.Sprintf("%s\nNo pairs to display.", title))
		return
	}

	var (
		chunks       []string
		currentChunk string
	)

	for i, pair := range pairs {
		line := fmt.Sprintf("%d %s: success: %d  failed: %d, profit: %.2f%%\n",
			i+1, pair.Key, pair.Result.Success, pair.Result.Fail, pair.Result.Ratio)
		currentChunk += line

		if (i+1)%chunkSize == 0 || i == len(pairs)-1 {
			chunks = append(chunks, currentChunk)
			currentChunk = ""
		}
	}

	for i, chunk := range chunks {
		header := title
		if i > 0 {
			header = fmt.Sprintf("%s (Page %d/%d)", title, i+1, len(chunks))
		}
		sendFunc(fmt.Sprintf("%s\n%s", header, chunk))
	}
}

func writeJSONAsyncWithTimestamp(prefix string, data interface{}) {
	go func() {
		timestamp := time.Now().Format("20060102_150405")
		filename := fmt.Sprintf("%s_%s.json", prefix, timestamp)

		file, err := os.Create(filename)
		if err != nil {
			log.Printf("创建文件失败: %v\n", err)
			return
		}
		defer file.Close()

		encoder := json.NewEncoder(file)
		encoder.SetIndent("", "  ")
		if err := encoder.Encode(data); err != nil {
			log.Printf("写入 JSON 失败: %v\n", err)
		} else {
			log.Printf("已写入 JSON 文件: %s\n", filename)
		}
	}()
}
