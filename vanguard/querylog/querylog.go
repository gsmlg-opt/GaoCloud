package querylog

import (
	"bytes"
	"cement/log"
	l4g "cement/log/log4go"
	"g53"
	"strconv"
	"sync"

	"vanguard/config"
	"vanguard/core"
)

const queryLogFormat = "%M"
const (
	logTrigger  = 200000
	logMessages = 100
)

type QueryLogger struct {
	core.DefaultHandler

	filelog  log.Logger
	tsNano   int64
	buf      *bytes.Buffer
	messages int
	logExt   bool
	lock     sync.Mutex
}

func NewQuerylog(conf *config.VanguardConf) core.DNSQueryHandler {
	q := &QueryLogger{buf: &bytes.Buffer{}}
	q.ReloadConfig(conf)
	return q
}

func (l *QueryLogger) ReloadConfig(conf *config.VanguardConf) {
	if l.filelog != nil {
		l.filelog.Close()
	}

	l.logExt = conf.Logger.Querylog.Extension
	if conf.Logger.Querylog.Path == "" {
		l.filelog = log.NewLog4jConsoleLoggerWithFmt(log.Info, l4g.NewDefaultFormater(queryLogFormat))
	} else {
		qlog, err := log.NewLog4jLoggerWithFmt(conf.Logger.Querylog.Path, log.Info,
			conf.Logger.Querylog.FileSize, conf.Logger.Querylog.Versions, l4g.NewDefaultFormater(queryLogFormat))
		if err != nil {
			panic("failed to create querylog" + err.Error())
		}
		l.filelog = qlog
		l.flushAndInit()
	}
}

func (l *QueryLogger) flushAndInit() {
	l.lock.Lock()
	if l.messages <= 0 {
		l.lock.Unlock()
		return
	}

	buf := l.buf
	l.buf = &bytes.Buffer{}
	l.messages = 0
	l.tsNano = 0
	l.lock.Unlock()

	l.filelog.Info(buf.String())
}

func (l *QueryLogger) optimalLogWrite(msg string, tsNano int64) {
	var buf *bytes.Buffer
	doLog := false

	l.lock.Lock()
	if tsNano-l.tsNano < logTrigger {
		l.buf.WriteString(msg)
		l.messages += 1
		if l.messages >= logMessages {
			buf = l.buf
			l.buf = &bytes.Buffer{}
			l.messages = 0
			doLog = true
		} else {
			l.buf.WriteString("\n")
		}
	} else {
		l.buf.WriteString(msg)
		buf = l.buf
		l.buf = &bytes.Buffer{}
		l.messages = 0
		doLog = true
	}
	l.tsNano = tsNano
	l.lock.Unlock()

	if doLog == true {
		l.filelog.Info(buf.String())
	}
}

func generateQueryExtension(delay int64, response *g53.Message) string {
	var msgBuffer bytes.Buffer

	msgBuffer.WriteString(" ")
	msgBuffer.WriteString(strconv.Itoa(int(delay)))
	if len(response.Sections[g53.AnswerSection]) == 0 {
		return msgBuffer.String()
	}

	msgBuffer.WriteString(" Response: ")
	for _, rrset := range response.Sections[g53.AnswerSection] {
		for _, rr := range rrset.Rdatas {
			msgBuffer.WriteString(rrset.Name.String(true))
			msgBuffer.WriteString(" ")
			msgBuffer.WriteString(rrset.Ttl.String())
			msgBuffer.WriteString(" ")
			msgBuffer.WriteString(rrset.Class.String())
			msgBuffer.WriteString(" ")
			msgBuffer.WriteString(rrset.Type.String())
			msgBuffer.WriteString(" ")
			msgBuffer.WriteString(rr.String())
			msgBuffer.WriteString(";")
		}
	}

	return msgBuffer.String()
}

func (l *QueryLogger) HandleQuery(ctx *core.Context) {
	core.PassToNext(l, ctx)
	l.LogWrite(ctx.Client)
}

func (l *QueryLogger) LogWrite(client core.Client) {
	var msgBuffer bytes.Buffer
	year, month, day, hour, min, sec, ms, tsNano := GetAllTimeArgs()
	delay := (tsNano - client.CreateTime.UnixNano()) / microSecToNanoSec

	msgBuffer.WriteString(Format2DigitNumber(day))
	msgBuffer.WriteString("-")
	msgBuffer.WriteString(GetMonthString(month))
	msgBuffer.WriteString("-")
	msgBuffer.WriteString(Format4DigitNumber(year))
	msgBuffer.WriteString(" ")
	msgBuffer.WriteString(Format2DigitNumber(hour))
	msgBuffer.WriteString(":")
	msgBuffer.WriteString(Format2DigitNumber(min))
	msgBuffer.WriteString(":")
	msgBuffer.WriteString(Format2DigitNumber(sec))
	msgBuffer.WriteString(".")
	msgBuffer.WriteString(Format3DigitNumber(ms))
	msgBuffer.WriteString(" client ")
	msgBuffer.WriteString(client.IP().String())
	msgBuffer.WriteString(" ")
	msgBuffer.WriteString(strconv.Itoa(client.Port()))
	msgBuffer.WriteString(": view ")
	msgBuffer.WriteString(client.View)
	msgBuffer.WriteString(": ")
	msgBuffer.WriteString(client.Request.Question.Name.String(true))
	msgBuffer.WriteString(" ")
	msgBuffer.WriteString(client.Request.Question.Class.String())
	msgBuffer.WriteString(" ")
	msgBuffer.WriteString(client.Request.Question.Type.String())
	if client.Response != nil {
		msgBuffer.WriteString(" ")
		msgBuffer.WriteString(client.Response.Header.Rcode.String())
		if set := client.Response.Header.GetFlag(g53.FLAG_RD); set {
			msgBuffer.WriteString(" +")
		} else {
			msgBuffer.WriteString(" -")
		}

		//no signature
		msgBuffer.WriteString(" NS")
		if client.Response.Edns != nil {
			msgBuffer.WriteString(" E")
		} else {
			msgBuffer.WriteString(" NE")
		}

		if client.UsingTCP {
			msgBuffer.WriteString(" T")
		} else {
			msgBuffer.WriteString(" NT")
		}

		//no do bit set
		msgBuffer.WriteString(" ND")
		if set := client.Response.Header.GetFlag(g53.FLAG_CD); set {
			msgBuffer.WriteString(" C")
		} else {
			msgBuffer.WriteString(" NC")
		}
		if client.Response.Header.GetFlag(g53.FLAG_AA) == false {
			if client.CacheHit == true {
				msgBuffer.WriteString(" H")
			} else {
				msgBuffer.WriteString(" NH")
			}
		} else {
			msgBuffer.WriteString(" A")
		}

		if l.logExt == true {
			msgBuffer.WriteString(generateQueryExtension(delay, client.Response))
		}
	}

	l.optimalLogWrite(msgBuffer.String(), tsNano)
}
