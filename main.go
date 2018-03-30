package main

import (
	"encoding/json"
	"flag"
	"log"
	"net/smtp"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"time"
	//使用了https://github.com/scorredoira/email中的email.go
	"github.com/email"
)

//传递参数
var (
	confile *string = flag.String("config", "email.conf", "-config 文件名")
	dir     *string = flag.String("dir", "tag.csv", "-dir 目录")
	tag     *string = flag.String("tag", "", "-tag a,b,c")
)

//用来解析email配置的
type Email struct {
	Title       string   `json:"title"`
	Message     string   `json:"message"`
	FromName    string   `json:"fromName"`
	FromAddress string   `json:"fromAddress"`
	To          []string `json:"to"`
	Filepath    string   `json:"filepath"`
	EmailServer string   `json:"emailserver"`
	EmailDome   string   `json:"Emaildome"`
	Username    string   `json:"username"`
	Password    string   `json:"password"`
}

var (
	conf   Email
	errtag []string
	tagc   []string
)

func init() {
	runtime.GOMAXPROCS(runtime.NumCPU())
}

func Error(err error) {
	if err != nil {
		log.Fatal(err)
	}
}

//用于发送邮件
func annex_email() {
	m := email.NewMessage(conf.Title, conf.Message)
	m.From.Name = conf.FromName
	m.From.Address = conf.FromAddress
	m.To = conf.To
	err := m.Attach(conf.Filepath)
	Error(err)
	err = email.Send(conf.EmailServer, smtp.PlainAuth("", conf.Username, conf.Password, conf.EmailDome), m)
	Error(err)
}

//从Python脚本用于获取异常IP
func readerr(f string, t string, c chan<- string) {
	p, err := exec.Command("python", "get_datav2.py", t, f).Output()
	if err != nil {
		c <- "0"
	} else {
		c <- string(p)
	}
}

func main() {
	tNow := time.Now()
	timeNow := tNow.Format("20060102")
	flag.Parse()

	log.Println("[INFO] ", "Working start!")

	c := make(chan string)
	taglist := strings.Split(*tag, ",")

	f1, err := os.Open(*confile)
	defer f1.Close()
	if err != nil {
		flag.Usage()
	}
    //解析邮件配置
	err = json.NewDecoder(f1).Decode(&conf)
	Error(err)

	for _, d := range taglist {
		go readerr(*dir, d, c)
		//获取Python脚本执行后的结果
		tag_num := <-c
		//脚本执行错误
		if tag_num == "0" {
			log.Println("[ERROR] ", d, " Parameter error may")
            os.Exit(3)
		}
		//接口故障
		if tag_num == "1\n" {
			log.Println("[ERROR] [", d, "] The interface data is empty!")
			errtag = append(errtag, d+": interface error\n")
			continue
		}
		//无错误值,故无返回
		if tag_num == "" {
			log.Println("[INFO] [", d, "] Warning ip is null")
			continue
		}
		log.Println("[INFO] [", d, "] on processing")
		//邮件内容
		tagc = append(tagc, tag_num)

	}
	//组装邮件正文
	errstr := strings.Join(errtag, "")
	tags := strings.Join(tagc, "")
	tagerr := "异常IP数量: \n" + tags + errstr + "\n\n" + "详情请查看附件\n"
	conf.Message = tagerr
	conf.Filepath = *dir + "/" + timeNow + "_tag" + ".csv"
	//发送邮件
	annex_email()
	log.Println("[INFO] ", "Send Mmail", " ok!")
}
