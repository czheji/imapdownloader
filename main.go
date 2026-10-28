package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/emersion/go-imap/client"
	log "github.com/sirupsen/logrus"
	"gopkg.in/yaml.v3"
)

func init() {
	// log.SetFormatter(&log.Formatter())
	log.SetOutput(os.Stdout)
	file, err := os.OpenFile("logrus.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err == nil {
		log.SetOutput(file)
	} else {
		log.Info("Failed to log to file, using default stderr")
	}
	log.SetLevel(log.TraceLevel)
}
func main() {
	configPath, outDir := parseFlags()
	log.Infof("imapdownloader %s\n", version)
	opts := &Options{}
	buf, err := os.ReadFile(configPath)
	if err != nil {
		log.Fatalf("读取配置文件出错(%s): %s\n", configPath, err.Error())
	}
	err = yaml.Unmarshal(buf, opts)
	if err != nil {
		log.Fatalf("转换配置文件出错(%s): %s\n", configPath, err.Error())
	}
	opts.setAbsDir(outDir)
	opts.print()
	ctx := context.Background()
	var runErr error
	switch opts.EffectiveProtocol() {
	case "pop3":
		runErr = DownloadByPOP3Account(ctx, opts)
	case "imap":
		runErr = DownloadByAccount(ctx, opts)
	default:
		log.Fatalf("不支持的协议：%s（仅支持 imap / pop3）", opts.Protocol)
	}
	if runErr != nil {
		log.Printf("下载报错：%s\n", runErr.Error())
	}
}

// parseFlags 解析命令行参数：
//   -c / --config  指定配置文件路径，缺省 config.yaml（与历史行为一致）
//   -o / --out     指定邮件导出根目录，缺空时沿用配置文件中的 dir（与历史行为一致）
//
// 拖拽支持：Windows 下把 config.yaml 拖到 imapdownloader.exe 上时，
// 文件路径会作为第一个位置参数传入，此时自动以该文件为配置、其所在目录为导出根目录。
func parseFlags() (configPath, outDir string) {
	flag.StringVar(&configPath, "c", "config.yaml", "配置文件路径（默认：config.yaml）")
	flag.StringVar(&configPath, "config", "config.yaml", "配置文件路径（同 -c）")
	flag.StringVar(&outDir, "o", "", "邮件导出根目录（缺省则使用配置文件中的 dir）")
	flag.StringVar(&outDir, "out", "", "邮件导出根目录（同 -o）")

	flag.Usage = func() {
		bin := filepath.Base(os.Args[0])
		fmt.Fprintf(os.Stderr, "imapdownloader %s\n\n", version)
		fmt.Fprintf(os.Stderr, "用法:\n")
		fmt.Fprintf(os.Stderr, "  %s [-c 配置文件] [-o 导出根目录]\n", bin)
		fmt.Fprintf(os.Stderr, "  %s config.yaml        # 也可直接把配置文件拖到本程序图标上运行\n\n", bin)
		fmt.Fprintf(os.Stderr, "参数:\n")
		flag.PrintDefaults()
		fmt.Fprintf(os.Stderr, "\n示例:\n")
		fmt.Fprintf(os.Stderr, "  %s                                  # 使用同目录 config.yaml，导出到配置的 dir\n", bin)
		fmt.Fprintf(os.Stderr, "  %s -c D:\\mail\\my.yaml -o D:\\mail\\backup\n", bin)
		fmt.Fprintf(os.Stderr, "  %s --config /etc/imap/my.yaml --out /data/mail\n", bin)
	}
	flag.Parse()

	explicit := map[string]bool{}
	flag.Visit(func(f *flag.Flag) { explicit[f.Name] = true })
	configPath, outDir = resolveDragDrop(configPath, outDir, explicit, flag.Args())
	return configPath, outDir
}

// resolveDragDrop 处理 Windows 拖拽启动：把拖入文件路径作为位置参数时，
// 以该文件为配置文件，其所在目录为导出根目录；显式命令行参数（-c/-o）优先。
func resolveDragDrop(configPath, outDir string, explicit map[string]bool, args []string) (string, string) {
	if len(args) == 0 {
		return configPath, outDir
	}
	dropped := args[0]
	if !explicit["c"] && !explicit["config"] {
		configPath = dropped
	}
	if !explicit["o"] && !explicit["out"] {
		if abs, err := filepath.Abs(dropped); err == nil {
			outDir = filepath.Dir(abs)
		}
	}
	return configPath, outDir
}

// DownloadByAccount 按邮箱账户进行下载
func DownloadByAccount(ctx context.Context, opts *Options) (err error) {
	d, err := NewDownloader(opts)
	if err != nil {
		return
	}

	defer func(Client *client.Client) {
		err := Client.Logout()
		if err != nil {
			log.Printf("退出登录出错：%s\n", err.Error())
		}
	}(d.Client)

	// 获取邮箱文件夹，并按前缀进行匹配
	mailboxes, err := d.getPrefixMatchedMailBoxes(ctx)

	// 逐个文件夹下载
	for _, mailbox := range mailboxes {
		err := d.downloadAccountMailbox(ctx, mailbox)
		if err != nil {
			return err
		}
	}

	return
}
