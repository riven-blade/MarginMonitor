package margin_monitor

import (
	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"
	"log"
	"margin_monitor/config"
	"os"
	"time"
)

func (p *Pair) HandleTopPair(pairs []pairEntry, bot config.Bot) ([]pairEntry, error) {
	topNum := bot.TopNum

	if topNum > len(pairs) {
		topNum = len(pairs)
	}

	if topNum <= 0 {
		return nil, nil
	}

	pairs = pairs[:topNum]

	topPairs := make([]string, topNum)
	for i := range topPairs {
		topPairs[i] = pairs[i].Key
	}

	// 加载配置
	configStr, err := LoadJSONFromFile(bot.TopConfigPath)
	if err != nil {
		log.Printf("failed to load config: %v", err)
		return pairs, err
	}

	// 读取现有 whitelist
	whitelist := gjson.Get(configStr, "exchange.pair_whitelist")
	var currentList []string
	for _, v := range whitelist.Array() {
		currentList = append(currentList, v.String())
	}

	// 对比是否需要更新
	if areStringSlicesEqualIgnoreOrder(currentList, topPairs) {
		log.Println("pair_whitelist unchanged, skipping update.")
		return pairs, err
	}

	// 覆盖写入新 whitelist（注意顺序为 topPairs 的顺序）
	updated, err := sjson.Set(configStr, "exchange.pair_whitelist", topPairs)
	if err != nil {
		log.Printf("failed to update pair_whitelist: %v", err)
		return pairs, err
	}

	if err := SaveJSONToFile(bot.TopConfigPath, updated); err != nil {
		log.Printf("failed to save updated config: %v", err)
		return pairs, err
	}

	log.Printf("pair_whitelist updated (%s): %v", bot.TopConfigPath, topPairs)
	time.Sleep(10 * time.Second)
	// 触发api 重启机器人 reload_config
	err = RestartBot(bot.ReloadTopApi, bot.Username, bot.Passwd)
	if err != nil {
		log.Printf("failed to restart bot: %v", err)
	}
	return pairs, err
}

func (p *Pair) HandlePair(pairs []pairEntry, bot config.Bot) ([]pairEntry, error) {
	if bot.TopNum >= len(pairs) {
		return nil, nil
	}
	pairs = pairs[bot.TopNum:]

	if bot.PairNum < len(pairs) {
		pairs = pairs[:bot.PairNum]
	}

	if len(pairs) == 0 {
		return nil, nil
	}

	seekPairs := make([]string, len(pairs))
	for i := range seekPairs {
		seekPairs[i] = pairs[i].Key
	}

	// 加载配置
	configStr, err := LoadJSONFromFile(bot.ConfigPath)
	if err != nil {
		log.Printf("failed to load config: %v", err)
		return pairs, err
	}

	// 读取现有 whitelist
	whitelist := gjson.Get(configStr, "exchange.pair_whitelist")
	var currentList []string
	for _, v := range whitelist.Array() {
		currentList = append(currentList, v.String())
	}

	// 对比是否需要更新
	if areStringSlicesEqualIgnoreOrder(currentList, seekPairs) {
		log.Println("pair_whitelist unchanged, skipping update.")
		return pairs, err
	}

	// 覆盖写入新 whitelist（注意顺序为 seekPairs 的顺序）
	updated, err := sjson.Set(configStr, "exchange.pair_whitelist", seekPairs)
	if err != nil {
		log.Printf("failed to update pair_whitelist: %v", err)
		return pairs, err
	}

	if err := SaveJSONToFile(bot.ConfigPath, updated); err != nil {
		log.Printf("failed to save updated config: %v", err)
		return pairs, err
	}

	log.Printf("pair_whitelist updated (%s): %v", bot.TopConfigPath, seekPairs)
	time.Sleep(10 * time.Second)
	// 触发api 重启机器人 reload_config
	err = RestartBot(bot.ReloadAPI, bot.Username, bot.Passwd)
	if err != nil {
		log.Printf("failed to restart bot: %v", err)
	}
	return pairs, err
}

func LoadJSONFromFile(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// SaveJSONToFile 保存 JSON 内容到文件
func SaveJSONToFile(path string, jsonStr string) error {
	return os.WriteFile(path, []byte(jsonStr), 0644)
}

// GetValueByPath 读取嵌套字段值
func GetValueByPath(jsonStr, path string) gjson.Result {
	return gjson.Get(jsonStr, path)
}

// SetValueByPath 设置嵌套字段值
func SetValueByPath(jsonStr, path string, value interface{}) (string, error) {
	return sjson.Set(jsonStr, path, value)
}

// DeleteValueByPath 删除嵌套字段值
func DeleteValueByPath(jsonStr, path string) (string, error) {
	return sjson.Delete(jsonStr, path)
}

// 比较两个 string slice 是否内容相同（忽略顺序）
func areStringSlicesEqualIgnoreOrder(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	setA := stringSliceToSet(a)
	setB := stringSliceToSet(b)
	for k := range setA {
		if _, ok := setB[k]; !ok {
			return false
		}
	}
	return true
}

// 转换 string slice 为 set（用于忽略顺序的比较）
func stringSliceToSet(slice []string) map[string]struct{} {
	set := make(map[string]struct{}, len(slice))
	for _, s := range slice {
		set[s] = struct{}{}
	}
	return set
}
