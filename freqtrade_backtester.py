#!/usr/bin/env python3
# -*- coding: utf-8 -*-
"""
freqtrade_backtester.py - 自动查找符合条件的交易对并进行回测，结果保存到Redis
使用freqtrade的pairlists机制进行交易对筛选，并下载最近一年的数据
"""
import os
import json
import argparse
import logging
import shutil
import time
from copy import deepcopy
from datetime import datetime, timedelta
from multiprocessing import Pool
import redis
import schedule

from freqtrade.commands import Arguments
from freqtrade.commands.optimize_commands import setup_optimize_configuration
from freqtrade.configuration import Configuration, validate_config_consistency, setup_utils_configuration
from freqtrade.data.dataprovider import DataProvider
from freqtrade.data.history.history_utils import download_data
from freqtrade.enums import RunMode
from freqtrade.optimize.backtesting import Backtesting
from freqtrade.resolvers import ExchangeResolver
from freqtrade.exchange import remove_exchange_credentials, Exchange

from freqtrade.plugins.pairlistmanager import PairListManager
from freqtrade.commands.data_commands import start_download_data, _check_data_config_download_sanity
from freqtrade.rpc import RPCManager

logging.basicConfig(
    level=logging.INFO,
    format='%(asctime)s - %(name)s - %(levelname)s - %(message)s',
)
logger = logging.getLogger('freqtrade_backtester')


class FreqtradeBacktester:
    """基于Freqtrade的自动回测工具，查找满足条件的交易对，结果保存到Redis"""

    def __init__(self, config_path: str, redis_url: str, ttl: int = 86400):
        """
        初始化回测工具

        Args:
            config_path: Freqtrade配置文件路径
            redis_url: Redis连接URL
            ttl: Redis数据过期时间(秒)，默认为1天
        """
        self.pairListsManager = None
        self.rpc = None
        self.dataprovider = None
        self.exchange = None
        self.freqtrade = None
        self.config = None
        self.config_path = config_path
        self.backtesting_config_path = "user_data/backtesting.json"
        self.redis_url = redis_url
        self.ttl = ttl
        self.redis_client = redis.from_url(redis_url)

    def get_candidate_pairs(self):
        """
        使用freqtrade的pairlists机制获取符合条件的交易对

        Returns:
            符合条件的交易对列表
        """
        logger.info("使用pairlists配置查找符合条件的交易对...")
        args = [
            "trade",
            "--config",
            self.config_path,
        ]
        self.config = Configuration(get_args(args), None).get_config()
        # Init the instance of the bot
        exchange_config = deepcopy(self.config["exchange"])
        # Remove credentials from original exchange config to avoid accidental credential exposure
        remove_exchange_credentials(self.config["exchange"], True)
        # Check config consistency here since strategies can set certain options
        validate_config_consistency(self.config)

        self.exchange = ExchangeResolver.load_exchange(
            self.config, exchange_config=exchange_config, load_leverage_tiers=True
        )

        self.rpc = RPCManager(self)
        self.dataprovider = DataProvider(self.config, self.exchange, rpc=self.rpc)
        self.pairListsManager = PairListManager(self.exchange, self.config, self.dataprovider)
        self.pairListsManager.refresh_pairlist()
        logger.info(f"找到 {len(self.pairListsManager.whitelist)} 个符合条件的交易对")

    def update_backtesting_config(self, whitelist: list[str]):
        try:
            with open(self.backtesting_config_path, "r", encoding="utf-8") as f:
                config = json.load(f)

            if "exchange" not in config:
                config["exchange"] = {}

            config["exchange"]["pair_whitelist"] = whitelist

            with open(self.backtesting_config_path, "w", encoding="utf-8") as f:
                json.dump(config, f, indent=4)

            logger.info(f"已将 {len(whitelist)} 个交易对写入配置文件。")

        except Exception as e:
            logger.error(f"更新 {self.backtesting_config_path} 文件失败: {e}")

    def run(self, days: int = 365):
        """
        执行整个流程：查找交易对、下载数据、回测并保存结果

        Args:
            days: 要下载的历史数据天数，默认365天
        """
        logger.info("开始执行自动回测流程...")
        try:
            # 查找符合条件的交易对
            self.get_candidate_pairs()
            whitelist = self.pairListsManager.whitelist

            # 下载数据
            if not whitelist:
                logger.error("没有找到交易对")
                return
            self.update_backtesting_config(whitelist)

            btc_pair = "BTC/USDT:USDT"
            # 先回测一下btc
            self.process_pair_backtesting(
                self.config["exchange"].get("name"),
                btc_pair,
                days,
                self.ttl
            )
            # 下载数据， 并回测
            for pair in whitelist:
                if pair.upper() == btc_pair.upper():
                    continue
                self.process_pair_backtesting(
                    self.config["exchange"].get("name"),
                    pair,
                    days,
                    self.ttl
                )
            logger.info("自动回测流程完成")

        except Exception as e:
            logger.error(f"执行过程中出错: {str(e)}")
            import traceback
            logger.error(traceback.format_exc())

        finally:
            data_dir = os.path.join("user_data", "data")
            if os.path.exists(data_dir):
                try:
                    shutil.rmtree(data_dir)
                    logger.info(f"已删除目录: {data_dir}")
                except Exception as e:
                    logger.error(f"删除目录 {data_dir} 时出错: {str(e)}")

    def download_data_pair(self, pair: str, timerange: str, days: int = 365) -> bool:
        """
        下载指定交易对的历史数据，针对每个币种单独下载。

        Args:
            days: 要下载的天数，默认365天（一年）
            pair: 交易对
            timerange:
        Returns:
            是否成功下载数据
        """
        logger.info(f"开始下载最近 {days} 天数据...")
        try:
            args = [
                "download-data",
                "--config", self.backtesting_config_path,
                "--timerange", timerange,
                "-t",
                "5m",
                "15m",
                "1h",
                "4h",
                "1d",
                "-p",
                pair
            ]

            # 使用 freqtrade 的下载函数
            config = setup_utils_configuration(get_args(args), RunMode.UTIL_EXCHANGE)
            _check_data_config_download_sanity(config)
            download_data(config, self.exchange)
            logger.info(f"{pair} 数据下载完成")
            return True

        except Exception as e:
            logger.error(f"{pair} 下载数据时出错: {str(e)}")
            return False  # ✅ 显式返回 False（避免 None）

    def process_pair_backtesting(self, name: str, pair: str, days: int,
                                 ttl: int = 86400):
        try:
            """
                    下载指定交易对的历史数据，针对每个币种单独下载。

                    Args:
                        pair: 交易对
                        days: 要下载的天数，默认365天（一年）

                    Returns:
                        是否成功下载数据
                    """
            logger.info(f"开始下载 {pair} 的最近 {days} 天数据...")
            end_date = datetime.now() + timedelta(hours=8)
            start_date = end_date - timedelta(days=days)
            timerange = f"{start_date.strftime('%Y%m%d')}-{end_date.strftime('%Y%m%d')}"
            download_data_result = self.download_data_pair(pair, timerange, days)
            if not download_data_result:
                logger.error(f"下载 {pair} 数据出错...")
                return
            logger.info(f"开始回测 {pair} 的最近 {days} 天数据...")

            timestamp = datetime.now().strftime("%Y%m%d_%H%M%S")
            export_filename = f"{pair.replace('/', '-').replace(':', '-')}_{timestamp}.zip"
            export_path = os.path.join("user_data", "backtest_results", export_filename)
            args = [
                "backtesting",
                "--config",
                self.backtesting_config_path,
                "--timerange",
                timerange,
                "-p",
                pair,
                "--export-filename",
                export_path,
                "--enable-protections"
            ]
            # 回测
            # Initialize configuration
            config = setup_optimize_configuration(get_args(args), RunMode.BACKTEST)
            logger.info("Starting freqtrade in Backtesting mode")
            # Initialize backtesting object
            backtesting = Backtesting(config, self.exchange)
            backtesting.start()
            strategy_comparison = backtesting.results.get("strategy_comparison")
            result = ""
            if len(strategy_comparison) > 0:
                strategy = strategy_comparison[0]
                wins = strategy['wins']
                losses = strategy['losses']
                profit_total_pct = strategy['profit_total_pct']
                result = f"{wins} {losses} {profit_total_pct}"
            logger.info(f"{pair} {strategy_comparison}")

            # 写入 Redis
            redis_key = f"{name}:backtest:{pair.replace('/', '_').replace(':', '_')}"
            self.redis_client.set(redis_key, result, ex=ttl)
            logger.info(f"{pair} 回测结果已写入 Redis，键: {redis_key}, 回测完成")
            return True
        except Exception as e:
            logger.error(f"处理 {pair} 时出错: {str(e)}")
            return False


def parse_args():
    """解析命令行参数"""
    parser = argparse.ArgumentParser(description='Freqtrade自动回测工具')
    parser.add_argument('-c', '--config', required=True, default='config.json', help='Freqtrade配置文件路径')
    parser.add_argument('-r', '--redis-url', default='redis://localhost:6379/0', help='Redis连接URL')
    parser.add_argument('-t', '--ttl', type=int, default=86400, help='Redis数据有效期(秒)')
    parser.add_argument('-d', '--days', type=int, default=365, help='下载历史数据的天数')
    return parser.parse_args()


def main():
    """主函数"""
    args = parse_args()

    logger.info("启动Freqtrade自动回测工具...")
    logger.info(f"配置文件: {args.config}")
    logger.info(f"Redis URL: {args.redis_url}")
    logger.info(f"数据有效期: {args.ttl}秒")
    logger.info(f"历史数据天数: {args.days}天")

    backtester = FreqtradeBacktester(
        config_path=args.config,
        redis_url=args.redis_url,
        ttl=args.ttl
    )

    backtester.run(days=args.days)


def get_args(args):
    return Arguments(args).get_parsed_arg()


if __name__ == "__main__":
    main()
    schedule.every(2).hours.do(main)
    logger.info("定时器已启动，每2小时执行一次 main()")
    while True:
        schedule.run_pending()
        time.sleep(1)
