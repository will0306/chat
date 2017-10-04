// WebChat project main.go
package main

import (
	"io"
    "fmt"
    "time"
	"model"
    "strings"
    "net/http"
	"crypto/md5"
    "encoding/json"
    "golang.org/x/net/websocket"
	"github.com/astaxie/beego/orm"
	_ "github.com/go-sql-driver/mysql"
)

//全局信息
var datas Datas
var users map[*websocket.Conn]string
var o orm.Ormer
var w_log io.Writer

func init() {
	orm.RegisterDriver("mysql", orm.DRMySQL)
	orm.RegisterDataBase("default", "mysql", "root:@/chat?charset=utf8")
	orm.RunSyncdb("default", false, true)
	orm.Debug = true
	o = orm.NewOrm()
	o.Using("default")
}


func main() {
    fmt.Println("启动时间")
    fmt.Println(time.Now())

    //初始化
    datas = Datas{}
    users = make(map[*websocket.Conn]string)

    //绑定效果页面
    http.HandleFunc("/", h_index)
    http.HandleFunc("/regist", regist)
    //绑定socket方法
    http.Handle("/webSocket", websocket.Handler(h_webSocket))
    //开始监听
    http.ListenAndServe(":8888", nil)
}

func h_index(w http.ResponseWriter, r *http.Request) {
    http.ServeFile(w, r, "login.html")
}

func regist(w http.ResponseWriter, r *http.Request){
	u := new(model.User)
	result := Response{Status : 1, Message : "注册成功", Data : ""}

	if r.FormValue("username") != "" {
		u.Name = r.FormValue("username")
	}
	if r.FormValue("password") != "" {
		u.Pwd = fmt.Sprintf("%x",md5.Sum([]byte(r.FormValue("password"))))
	}
	_, err := o.Insert(u)
	if err != nil {
		fmt.Println(err)
		result.Status = 0
		result.Message = "注册失败,用户名已存在"
	}
	
    b, errMarshl := json.Marshal(result)
    if errMarshl != nil {
        fmt.Println("结果处理异常...")
		w.Write([]byte("error"))
		return
    }
	u2 := new(model.User)
	qs := o.QueryTable("user")
	fmt.Println(qs.Filter("name", u.Name).One(u2))
	fmt.Println(u2.Name)
	w.Write(b)
}

func h_webSocket(ws *websocket.Conn) {


	//----------验证用户----------- start
	username := ws.Request().FormValue("username")
	pwd := fmt.Sprintf("%x",md5.Sum([]byte(ws.Request().FormValue("password"))))
	u := new(model.User)
	qs := o.QueryTable("user")
	if err := qs.Filter("name", username).One(u); err != nil || u.Pwd != pwd || u.Name == "" {
		var err_datas Datas
		err_Msg := UserMsg{UserName : "System", DataType : "send"}
		if u.Name == "" {
			err_Msg.Msg = "用户不存在"
		} else if u.Pwd != pwd {
			err_Msg.Msg = "密码错误"
		}
        err_datas.UserMsgs = make([]UserMsg, 0)
		err_datas.UserMsgs = append(err_datas.UserMsgs, err_Msg)
        err_b, errMarshl := json.Marshal(err_datas)
        if errMarshl != nil {
            fmt.Println("全局消息内容异常...")
			return
        }
		websocket.Message.Send(ws, string(err_b))
		ws.Close()
		return
	}
	//----------验证用户----------- end

    var userMsg UserMsg
    var data string
    for {

        //判断是否重复连接
        if _, ok := users[ws]; !ok {
            users[ws] = "匿名"
        }
        userMsgsLen := len(datas.UserMsgs)
        fmt.Println("UserMsgs", userMsgsLen, "users长度：", len(users))

        //有消息时，全部分发送数据
        if userMsgsLen > 0 {
            b, errMarshl := json.Marshal(datas)
            if errMarshl != nil {
                fmt.Println("全局消息内容异常...")
                break
            }
            for key, _ := range users {
                errMarshl = websocket.Message.Send(key, string(b))
                if errMarshl != nil {
                    //移除出错的链接
                    delete(users, key)
                    fmt.Println(users[key],"发送出错...")
                    break
                }
            }
            datas.UserMsgs = make([]UserMsg, 0)
        }

        fmt.Println("开始解析数据...")
        err := websocket.Message.Receive(ws, &data)
        fmt.Println("data：", data)
        if err != nil {
            //移除出错的链接
            fmt.Println("接收出错...")
            delete(users, ws)
			print(users[ws],"已被移除")
            break
        }

        data = strings.Replace(data, "\n", "", 0)
        err = json.Unmarshal([]byte(data), &userMsg)
        if err != nil {
            fmt.Println("解析数据异常...")
            break
        }
        fmt.Println("请求数据类型：", userMsg.DataType)

        switch userMsg.DataType {
        case "send":
            //赋值对应的昵称到ws
            if _, ok := users[ws]; ok {
                users[ws] = userMsg.UserName

                //清除连接人昵称信息
                datas.UserDatas = make([]UserData, 0)
                //重新加载当前在线连接人
                for _, item := range users {

                    userData := UserData{UserName: item}
                    datas.UserDatas = append(datas.UserDatas, userData)
                }
            }
            datas.UserMsgs = append(datas.UserMsgs, userMsg)
        }
    }

}

type UserMsg struct {
    UserName string
    Msg      string
    DataType string
}

type UserData struct {
    UserName string
}

type Datas struct {
    UserMsgs  []UserMsg
    UserDatas []UserData
}

type Response struct {
	Status int			`json:"status"`
	Message string		`json:"message"`
	Data interface{}	`json:"data"` 
}
