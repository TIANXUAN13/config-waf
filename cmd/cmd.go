package main

import (
	"bufio"
	"errors"
	"fmt"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/urfave/cli/v2"

	"safeline/api"
)

var modeMap = map[string]string{
	"1": "TransparentBridge",
	"2": "TransparentProxy",
	"3": "HardwareReverseProxy",
	"4": "SoftwareReverseProxy",
	"5": "Other",

	"TransparentBridge":    "1",
	"TransparentProxy":     "2",
	"HardwareReverseProxy": "3",
	"SoftwareReverseProxy": "4",
}

func cmdLs(nameSpace string, tokens *[][]string, n *time.Duration) *cli.Command {
	funcMap := map[string]func(*api.API, *os.File, bool, ...interface{}){
		"website":    DisplayWebsite,
		"policy":     DisplayPolicyGroup,
		"ipgroup":    DisplayIpGroup,
		"cert":       DisplaySSLCert,
		"acl":        DisplayAclRule,
		"grule":      DisplayGlobalRule,
		"crule":      DisplayCustomRule,
		"assetgroup": DisplayAssetGroup,
	}
	return &cli.Command{
		Name:  "ls",
		Usage: fmt.Sprintf("list %s", nameSpace),
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "output",
				Aliases: []string{"o"},
				Usage:   "write to `FILE` instead of stdout",
			}, &cli.BoolFlag{
				Name:    "raw",
				Aliases: []string{"r"},
				Usage:   "return json",
			},
		},
		Action: func(ctx *cli.Context) error {
			var w []*os.File
			saveName := ctx.String("output")
			raw := ctx.Bool("raw")
			for _, t := range *tokens {
				if saveName != "" {
					format := "csv"
					if raw {
						format = "json"
					}
					u, _ := url.Parse(t[0])
					name := fmt.Sprintf("%s_%s.%s", saveName, u.Hostname(), format)
					f, err := os.OpenFile(name, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
					panicIf(err)
					w = append(w, f)
				} else {
					w = append(w, os.Stdout)
				}
			}
			if saveName != "" {
				defer func() {
					for _, v := range w {
						v.Close()
					}
				}()
			}
			for idx, t := range *tokens {
				client := api.NewWithTimeout(t[0], t[1], "", *n)
				fmt.Printf(">>> %s\n", t[0])
				funcMap[nameSpace](client, w[idx], raw, modeMap[strip(t[2])])
				fmt.Println("<<<")
			}
			return nil
		},
	}
}

func cmdAddIp(nameSpace string, tokens *[][]string, n *time.Duration) *cli.Command {
	byNameFunc := map[string]func(*api.API, string, ...string) ([]byte, error){
		"ipgroup": AddIpByGroupName,
		"acl":     AddIpByAclRuleName,
	}
	byIdFunc := map[string]func(*api.API, int, ...string) ([]byte, error){
		"ipgroup": AddIpByGroupId,
		"acl":     AddIpByAclRuleId,
	}
	return &cli.Command{
		Name:  "addip",
		Usage: "添加IP",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:  "byname",
				Usage: "通过`名称`定位",
			}, &cli.IntFlag{
				Name:  "byid",
				Usage: "通过`ID`定位",
			}, &cli.StringFlag{
				Name:    "load_file",
				Aliases: []string{"f"},
				Usage:   "从文件中添加IP（每行一个IP）",
			},
		},
		Before: func(ctx *cli.Context) error {
			if ctx.String("byname") == "" && ctx.Int("byid") == 0 {
				return errors.New("required flags \"byname\" or \"byid\" not set")
			}
			return nil
		},
		Action: func(ctx *cli.Context) error {
			name := ctx.String("byname")
			id := ctx.Int("byid")
			filename := ctx.String("load_file")

			var target []string
			if filename != "" {
				f, err := os.Open(filename)
				if err != nil {
					panic(err)
				}
				defer f.Close()
				scanner := bufio.NewScanner(f)
				for scanner.Scan() {
					line := scanner.Text()
					line = strings.Trim(line, " ")
					if line != "" {
						target = append(target, line)
					}
				}
			} else {
				target = ctx.Args().Slice()
			}

			var b []byte
			var err error
			for _, t := range *tokens {
				client := api.NewWithTimeout(t[0], t[1], "", *n)
				if id != 0 {
					b, err = byIdFunc[nameSpace](client, id, target...)
				} else {
					b, err = byNameFunc[nameSpace](client, name, target...)
				}
				if ok, err := api.OK2(b, err); !ok {
					_, _ = fmt.Fprintf(os.Stderr, "[error] %v\n", err.Error())
				}
			}
			return nil
		},
	}
}

func cmdDelIp(nameSpace string, tokens *[][]string, n *time.Duration) *cli.Command {
	Func := map[string]func(*api.API, int, ...string) ([]byte, error){
		"ipgroup": DelIpByIpGroupId,
		"acl":     DelIpByAclRuleId,
	}
	return &cli.Command{
		Name:  "delip",
		Usage: "删除IP",
		Flags: []cli.Flag{
			&cli.IntFlag{
				Name:     "byid",
				Usage:    "通过`ID`定位",
				Required: true,
			},
		},
		Before: func(ctx *cli.Context) error {
			if ctx.Args().Len() < 1 {
				return errors.New("target IP is required")
			}
			return nil
		},
		Action: func(ctx *cli.Context) error {
			id := ctx.Int("byid")
			ips := ctx.Args().Slice()
			for _, t := range *tokens {
				client := api.NewWithTimeout(t[0], t[1], "", *n)
				fmt.Printf(">>> %s\n", t[0])
				_, err := Func[nameSpace](client, id, ips...)
				if err != nil {
					_, _ = fmt.Fprintf(os.Stderr, "[error] %v\n", err.Error())
					continue
				}
				fmt.Println("<<<")
			}
			return nil
		},
	}
}

func cmdGetIp(tokens *[][]string, n *time.Duration) *cli.Command {
	return &cli.Command{
		Name:  "getip",
		Usage: "查询IP",
		Flags: []cli.Flag{
			&cli.IntFlag{
				Name:     "byid",
				Usage:    "通过`ID`定位",
				Required: true,
			},
		},
		Action: func(ctx *cli.Context) error {
			id := ctx.Int("byid")
			for _, t := range *tokens {
				client := api.NewWithTimeout(t[0], t[1], "", *n)
				fmt.Printf(">>> %s\n", t[0])
				DisplayIpByAclRuleId(client, id)
				fmt.Println("<<<")
			}
			return nil
		},
	}
}

func cmdGetIpByIpGroup(tokens *[][]string, n *time.Duration) *cli.Command {
	return &cli.Command{
		Name:  "getip",
		Usage: "查询IP组内IP",
		Flags: []cli.Flag{
			&cli.IntFlag{
				Name:     "byid",
				Usage:    "通过`ID`定位",
				Required: true,
			},
		},
		Action: func(ctx *cli.Context) error {
			id := ctx.Int("byid")
			for _, t := range *tokens {
				client := api.NewWithTimeout(t[0], t[1], "", *n)
				fmt.Printf(">>> %s\n", t[0])
				if err := DisplayIpByIpGroupId(client, id); err != nil {
					_, _ = fmt.Fprintf(os.Stderr, "[error] %v\n", err.Error())
				}
				fmt.Println("<<<")
			}
			return nil
		},
	}
}

func cmdAllIpByIpGroup(tokens *[][]string, n *time.Duration) *cli.Command {
	return &cli.Command{
		Name:  "allip",
		Usage: "联合查询所有IP组及IP",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "output",
				Aliases: []string{"o"},
				Usage:   "write to `FILE` instead of stdout",
			}, &cli.BoolFlag{
				Name:    "raw",
				Aliases: []string{"r"},
				Usage:   "return json",
			},
		},
		Action: func(ctx *cli.Context) error {
			var w []*os.File
			saveName := ctx.String("output")
			raw := ctx.Bool("raw")
			for _, t := range *tokens {
				if saveName != "" {
					format := "csv"
					if raw {
						format = "json"
					}
					u, _ := url.Parse(t[0])
					name := fmt.Sprintf("%s_%s.%s", saveName, u.Hostname(), format)
					f, err := os.OpenFile(name, os.O_CREATE|os.O_WRONLY, 0644)
					panicIf(err)
					w = append(w, f)
				} else {
					w = append(w, os.Stdout)
				}
			}
			if saveName != "" {
				defer func() {
					for _, v := range w {
						v.Close()
					}
				}()
			}
			for idx, t := range *tokens {
				client := api.NewWithTimeout(t[0], t[1], "", *n)
				fmt.Printf(">>> %s\n", t[0])
				if err := DisplayIpGroupAllIp(client, w[idx], raw); err != nil {
					_, _ = fmt.Fprintf(os.Stderr, "[error] %v\n", err.Error())
				}
				fmt.Println("<<<")
			}
			return nil
		},
	}
}

func cmdExportIpByIpGroup(tokens *[][]string, n *time.Duration) *cli.Command {
	return &cli.Command{
		Name:  "exportip",
		Usage: "导出IP组及IP明细",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:     "filename",
				Aliases:  []string{"f"},
				Usage:    "保存的文件名（不含后缀）",
				Required: true,
			}, &cli.StringFlag{
				Name:  "format",
				Usage: "导出格式（csv/json）",
				Value: "csv",
			},
		},
		Action: func(ctx *cli.Context) error {
			filename := ctx.String("filename")
			format := ctx.String("format")
			for _, t := range *tokens {
				client := api.NewWithTimeout(t[0], t[1], "", *n)
				fmt.Printf(">>> %s\n", t[0])
				if err := ExportIpGroupIP(client, filename, format); err != nil {
					_, _ = fmt.Fprintf(os.Stderr, "[error] %v\n", err.Error())
				}
				fmt.Println("<<<")
			}
			return nil
		},
	}
}

func cmdDelAllIp(tokens *[][]string, n *time.Duration) *cli.Command {
	return &cli.Command{
		Name:  "clearip",
		Usage: "清空所有IP",
		Flags: []cli.Flag{
			&cli.IntFlag{
				Name:     "byid",
				Usage:    "通过`ID`定位",
				Required: true,
			},
		},
		Action: func(ctx *cli.Context) error {
			id := ctx.Int("byid")
			for _, t := range *tokens {
				client := api.NewWithTimeout(t[0], t[1], "", *n)
				fmt.Printf(">>> %s\n", t[0])
				b, err := DelAllIpByAclRuleId(client, id)
				if ok, err := api.OK2(b, err); !ok {
					_, _ = fmt.Fprintf(os.Stderr, "[error] %v\n", err.Error())
					continue
				}
				fmt.Println("<<<")
			}
			return nil
		},
	}
}

func cmdCreateAclRule(tokens *[][]string, n *time.Duration) *cli.Command {
	return &cli.Command{
		Name:  "create",
		Usage: "创建频率控制规则",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:     "name",
				Usage:    "规则名称",
				Required: true,
			}, &cli.IntFlag{
				Name:        "time",
				Usage:       "封禁时间（单位：s）",
				Value:       1296000, // 15天
				DefaultText: "1296000=15天",
			},
		},
		Before: func(ctx *cli.Context) error {
			if ctx.Args().Len() < 1 {
				return errors.New("target or IP Group is required")
			}
			return nil
		},
		Action: func(ctx *cli.Context) error {
			name := ctx.String("name")
			expire := ctx.Int("time")
			targets := ctx.Args().Slice()
			for _, t := range *tokens {
				client := api.NewWithTimeout(t[0], t[1], "", *n)
				fmt.Printf(">>> %s\n", t[0])
				_, err := CreateAclRule(client, name, expire, targets...)
				if err != nil {
					_, _ = fmt.Fprintf(os.Stderr, "[error] %v\n", err.Error())
					continue
				}
				fmt.Println("<<<")
			}
			return nil
		},
	}
}

func cmdDeleteAclRule(tokens *[][]string, n *time.Duration) *cli.Command {
	return &cli.Command{
		Name:  "delete",
		Usage: "删除频率控制规则",
		Before: func(ctx *cli.Context) error {
			if ctx.Args().Len() < 1 {
				return errors.New("rule id is required")
			}
			return nil
		},
		Action: func(ctx *cli.Context) error {
			sl := ctx.Args().Slice()
			var ids []int
			for _, v := range sl {
				id, err := strconv.Atoi(v)
				if err != nil {
					return err
				}
				ids = append(ids, id)
			}
			for _, t := range *tokens {
				client := api.NewWithTimeout(t[0], t[1], "", *n)
				fmt.Printf(">>> %s\n", t[0])
				_, err := DeleteAclRule(client, ids...)
				if err != nil {
					_, _ = fmt.Fprintf(os.Stderr, "[error] %v\n", err.Error())
					continue
				}
				fmt.Println("<<<")
			}
			return nil
		},
	}
}

func cmdCreateIpGroup(tokens *[][]string, n *time.Duration) *cli.Command {
	return &cli.Command{
		Name:  "create",
		Usage: "创建IP组",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:     "name",
				Usage:    "工作组名称",
				Required: true,
			}, &cli.StringFlag{
				Name:  "comment",
				Usage: "备注",
			},
		},
		Before: func(ctx *cli.Context) error {
			if ctx.Args().Len() < 1 {
				return errors.New("target IP is required")
			}
			return nil
		},
		Action: func(ctx *cli.Context) error {
			name := ctx.String("name")
			comment := ctx.String("comment")
			targets := ctx.Args().Slice()
			for _, t := range *tokens {
				client := api.NewWithTimeout(t[0], t[1], "", *n)
				fmt.Printf(">>> %s\n", t[0])
				_, err := CreateIpGroup(client, name, comment, targets...)
				if err != nil {
					_, _ = fmt.Fprintf(os.Stderr, "[error] %v\n", err.Error())
					continue
				}
				fmt.Println("<<<")
			}
			return nil
		},
	}
}

func cmdDeleteIpGroup(tokens *[][]string, n *time.Duration) *cli.Command {
	return &cli.Command{
		Name:  "delete",
		Usage: "删除IP组",
		Before: func(ctx *cli.Context) error {
			if ctx.Args().Len() < 1 {
				return errors.New("ipgroup id is required")
			}
			return nil
		},
		Action: func(ctx *cli.Context) error {
			sl := ctx.Args().Slice()
			var ids []int
			for _, v := range sl {
				id, err := strconv.Atoi(v)
				if err != nil {
					return err
				}
				ids = append(ids, id)
			}
			for _, t := range *tokens {
				client := api.NewWithTimeout(t[0], t[1], "", *n)
				fmt.Printf(">>> %s\n", t[0])
				_, err := DeleteIpGroup(client, ids...)
				if err != nil {
					_, _ = fmt.Fprintf(os.Stderr, "[error] %v\n", err.Error())
					continue
				}
				fmt.Println("<<<")
			}
			return nil
		},
	}
}

func cmdCreateWebsite(tokens *[][]string, n *time.Duration) *cli.Command {
	return &cli.Command{
		Name:  "create",
		Usage: "创建站点",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:     "filename",
				Aliases:  []string{"f"},
				Usage:    "模板文件",
				Required: true,
			},
		},
		Action: func(ctx *cli.Context) error {
			filename := ctx.String("filename")
			globalMode := ctx.String("mode")
			mode := globalMode
			for _, t := range *tokens {
				client := api.NewWithTimeout(t[0], t[1], "", *n)
				fmt.Printf(">>> %s\n", t[0])
				if globalMode == "" {
					mode = modeMap[strip(t[2])]
				}
				CreateWebsite(client, filename, mode)
				fmt.Println("<<<")
			}
			return nil
		},
	}
}

func cmdUpdateWebsite(tokens *[][]string, n *time.Duration) *cli.Command {
	var limit []int
	return &cli.Command{
		Name:  "update",
		Usage: "更新站点",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:     "filename",
				Aliases:  []string{"f"},
				Required: true,
				Usage:    "模板文件",
			}, &cli.StringFlag{
				Name:     "select",
				Aliases:  []string{"s"},
				Required: true,
				Usage: `选择更新的字段 e.g. "0,1,2"
	0：站点是否启用（21及以上版本支持）
	1：防护策略ID
	2：源IP获取方式
	3：SSL证书ID
	4：工作组（透明代理与硬件反代模式）
	5：web资产组ID
	10：访问日志设置 (透明代理与软、硬件反代模式）
	
	以下仅支持软件反代与硬件反代模式：
	6：站点生效URL路径（23版本开始支持）
	7：站点生效的检测节点（23版本开始支持）
	8：用户识别方式
	9：回源IP配置
	11：HTTP头配置
	12：保持连接配置
	13：业务服务器获取源IP XFF 配置`,
			},
		},
		Before: func(ctx *cli.Context) error {
			col := ctx.String("select")
			sl := stringToList(strip(col))
			for _, v := range sl {
				limit = append(limit, stringToInt(v))
			}
			return nil
		},
		Action: func(ctx *cli.Context) error {
			filename := ctx.String("filename")
			globalMode := ctx.String("mode")
			mode := globalMode
			for _, t := range *tokens {
				client := api.NewWithTimeout(t[0], t[1], "", *n)
				fmt.Printf(">>> %s\n", t[0])
				if globalMode == "" {
					mode = modeMap[strip(t[2])]
				}
				UpdateWebsite(client, filename, mode, limit...)
				fmt.Println("<<<")
			}
			return nil
		},
	}
}

func cmdExportWebsite(tokens *[][]string, n *time.Duration) *cli.Command {
	return &cli.Command{
		Name:  "export",
		Usage: "导出模版",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:     "filename",
				Aliases:  []string{"f"},
				Usage:    "保存的文件名",
				Required: true,
			},
		},
		Action: func(ctx *cli.Context) error {
			filename := ctx.String("filename")
			globalMode := ctx.String("mode")
			mode := globalMode
			for _, t := range *tokens {
				client := api.NewWithTimeout(t[0], t[1], "", *n)
				fmt.Printf(">>> %s\n", t[0])
				if globalMode == "" {
					mode = modeMap[strip(t[2])]
				}
				ExportWebsite(client, filename, mode)
				fmt.Println("<<<")
			}
			return nil
		},
	}
}

func cmdImportRule(tokens *[][]string, n *time.Duration, global bool) *cli.Command {
	return &cli.Command{
		Name:  "import",
		Usage: "导入规则",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:     "filename",
				Aliases:  []string{"f"},
				Usage:    "json文件名",
				Required: true,
			},
		},
		Action: func(ctx *cli.Context) error {
			filename := ctx.String("filename")
			for _, t := range *tokens {
				fmt.Printf(">>> %s\n", t[0])
				client := api.NewWithTimeout(t[0], t[1], "", *n)
				ImportRule(client, filename, global)
				fmt.Println("<<<")
			}
			return nil
		},
	}
}

func cmdExportRule(tokens *[][]string, n *time.Duration, global bool) *cli.Command {
	return &cli.Command{
		Name:  "export",
		Usage: "导出规则",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:     "filename",
				Aliases:  []string{"f"},
				Usage:    "json文件名",
				Required: true,
			},
		},
		Action: func(ctx *cli.Context) error {
			filename := ctx.String("filename")
			for _, t := range *tokens {
				fmt.Printf(">>> %s\n", t[0])
				client := api.NewWithTimeout(t[0], t[1], "", *n)
				ExportRule(client, filename, global)
				fmt.Println("<<<")
			}
			return nil
		},
	}
}

func main() {
	var tokens [][]string
	var timeout = new(time.Duration)
	app := cli.App{
		Name:                 "safeline",
		Usage:                "雷池命令行工具",
		Version:              "2024.11.11",
		EnableBashCompletion: true,
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "config",
				Aliases: []string{"c"},
				Usage:   "配置文件，默认加载当前目录下的 token.csv 文件",
				Value:   "token.csv",
			}, &cli.StringFlag{
				Name:  "token",
				Usage: "雷池 openapi token",
			}, &cli.StringFlag{
				Name:  "addr",
				Usage: "雷池管理节点地址，比如:\"https://169.254.0.4:1443\"",
			}, &cli.StringFlag{
				Name:    "match",
				Aliases: []string{"m"},
				Usage:   "当配置文件中有多个雷池管理节点 url，使用这个参数可匹配需要的地址，匹配关系为字符串包含",
			}, &cli.BoolFlag{
				Name:    "debug",
				Aliases: []string{"d"},
				Usage:   "更新站点与创建站点时，输出解析 csv 模版后的数据",
			}, &cli.Int64Flag{
				Name:        "timeout",
				Value:       5,
				DefaultText: "5",
				Aliases:     []string{"t"},
				Usage:       "单个请求超时时间，单位为 s",
			},
		},
		Before: func(ctx *cli.Context) error {
			DEBUG = ctx.Bool("debug")
			addr := ctx.String("addr")
			token := ctx.String("token")
			n := ctx.Int64("timeout")
			*timeout = time.Duration(n) * time.Second
			if ctx.Args().First() == "web" {
				return nil
			}
			if addr != "" && token != "" {
				tokens = [][]string{{addr, token, "5"}}
			} else {
				ret := loadToken(ctx.String("config"))
				pattern := ctx.String("match")
				if pattern != "" {
					for _, t := range ret {
						if strings.Contains(t[0], pattern) {
							tokens = append(tokens, t)
						}
					}
				} else {
					tokens = ret
				}
			}
			return nil
		},
		Commands: []*cli.Command{
			cmdWeb(),
			{
				Name:  "website",
				Usage: "防护站点",
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:  "mode",
						Usage: "部署模式[TransparentBridge TransparentProxy HardwareReverseProxy SoftwareReverseProxy]",
						Value: "",
					},
				},
				Before: func(ctx *cli.Context) error {
					token := ctx.String("token")
					addr := ctx.String("addr")
					mode := ctx.String("mode")
					if token != "" && addr != "" && mode != "" {
						tokens[0][2] = modeMap[mode]
					}
					return nil
				},
				Subcommands: []*cli.Command{
					cmdLs("website", &tokens, timeout),
					cmdCreateWebsite(&tokens, timeout),
					cmdUpdateWebsite(&tokens, timeout),
					cmdExportWebsite(&tokens, timeout),
				},
			}, {
				Name:  "policy",
				Usage: "防护策略",
				Subcommands: []*cli.Command{
					cmdLs("policy", &tokens, timeout),
				},
			}, {
				Name:  "ipgroup",
				Usage: "IP组",
				Subcommands: []*cli.Command{
					cmdLs("ipgroup", &tokens, timeout),
					cmdGetIpByIpGroup(&tokens, timeout),
					cmdAllIpByIpGroup(&tokens, timeout),
					cmdExportIpByIpGroup(&tokens, timeout),
					cmdAddIp("ipgroup", &tokens, timeout),
					cmdDelIp("ipgroup", &tokens, timeout),
					cmdCreateIpGroup(&tokens, timeout),
					cmdDeleteIpGroup(&tokens, timeout),
				},
			}, {
				Name:  "cert",
				Usage: "SSL证书",
				Subcommands: []*cli.Command{
					cmdLs("cert", &tokens, timeout),
				},
			}, {
				Name:  "acl",
				Usage: "频率控制",
				Subcommands: []*cli.Command{
					cmdLs("acl", &tokens, timeout),
					cmdAddIp("acl", &tokens, timeout),
					cmdGetIp(&tokens, timeout),
					cmdDelIp("acl", &tokens, timeout),
					cmdDelAllIp(&tokens, timeout),
					cmdCreateAclRule(&tokens, timeout),
					cmdDeleteAclRule(&tokens, timeout),
				},
			}, {
				Name:  "global-rule",
				Usage: "全局自定义规则",
				Subcommands: []*cli.Command{
					cmdLs("grule", &tokens, timeout),
					cmdImportRule(&tokens, timeout, true),
					cmdExportRule(&tokens, timeout, true),
				},
			}, {
				Name:  "custom-rule",
				Usage: "站点自定义规则",
				Subcommands: []*cli.Command{
					cmdLs("crule", &tokens, timeout),
					cmdImportRule(&tokens, timeout, false),
					cmdExportRule(&tokens, timeout, false),
				},
			}, {
				Name:  "assetgroup",
				Usage: "web资产组",
				Subcommands: []*cli.Command{
					cmdLs("assetgroup", &tokens, timeout),
				},
			},
		},
	}
	if err := app.Run(os.Args); err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "[error] %v\n", err.Error())
	}
}
