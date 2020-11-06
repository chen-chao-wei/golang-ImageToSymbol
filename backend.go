package main

import (
    "fmt"
	"log"
	"net/http"
	"html/template"
	"io"
	"io/ioutil"
	"os"
	"path"
	"errors"
    "strings"
    "image"
    "github.com/nfnt/resize"
    "strconv"
    "time"
)
import _ "image/jpeg"
import _ "image/png"

type IndexData struct {
	Title string
    Content string
    Filename string
    Error string
    Result string
    GoroutineResult string
}
const pageTitle = "將上傳圖片轉成文字符號"
const pageContent ="寬度若大於300像素，會等比壓縮至300像素"
var symbol = [16]string{" ",".","-","^","~","!","(",")","=","%","&","$","/","*","#","@" } 
    
func indexPage(w http.ResponseWriter, r *http.Request)  {
	tmpl := template.Must(template.ParseFiles("./index.html"))
	data:=new(IndexData)
	data.Title = pageTitle
	data.Content =pageContent
	tmpl.Execute(w,data)
}
func frontJS(w http.ResponseWriter, r *http.Request)  {
	tmpl := template.Must(template.ParseFiles("./front.js"))
	
	tmpl.Execute(w,"null")
}
// 上传图像接口
func uploadHandle (w http.ResponseWriter, req *http.Request) {
    w.Header().Set("Content-Type", "text/html")
 
    req.ParseForm()
    if req.Method != "POST" {
        tmpl := template.Must(template.ParseFiles("./index.html"))
	    data:=new(IndexData)
	    data.Title = pageTitle
	    data.Content =pageContent
	    tmpl.Execute(w,data)
    } else {
        // 接收图片
        uploadFile, handle, err := req.FormFile("image")
        fmt.Println(err)
        if err != nil{
            errorHandle(err, w)
            return
        }
        
        // 检查图片后缀
        ext := strings.ToLower(path.Ext(handle.Filename))
        if ext != ".jpg" && ext != ".png" {
            errorHandle(errors.New("只支援jpg/png檔案格式"), w);
            return
            //defer os.Exit(2)
        }
        
		saveFileName := []byte(handle.Filename)
        fmt.Println(string(saveFileName))
        // 保存图片
        os.Mkdir("./uploaded/", 0777)
        saveFile, err := os.OpenFile("./uploaded/" + string(saveFileName) , os.O_WRONLY|os.O_CREATE, 0666);
        errorHandle(err, w)
        io.Copy(saveFile, uploadFile);
 
        defer uploadFile.Close()
        defer saveFile.Close()
        // 上传图片成功
        tmpl := template.Must(template.ParseFiles("./uploaded.html"))
	    data:=new(IndexData)
	    data.Title = pageTitle
        data.Content =pageContent
        data.Filename =handle.Filename
        // w.Write([]byte("查看上傳圖片: <a target='_blank' href='/uploaded/" + handle.Filename + "'>" + handle.Filename + "</a>"));
        data.Result = imageToSymbol("./uploaded/"+string(saveFileName),w)
        data.GoroutineResult = imageToSymbol2("./uploaded/"+string(saveFileName),w)
        
        tmpl.Execute(w,data)
    }
}
func imageToSymbol(name string, web http.ResponseWriter)  string{
    t1:=time.Now()
    fmt.Println(name)
    file, err := os.Open(name)
    errorHandle(err, web);
    defer file.Close()
    imageDecode, _, err := image.Decode(file)
    if imageDecode.Bounds().Dx()>300 {
        imageDecode = resize.Resize(300,0,imageDecode,resize.Lanczos3)
    }
    
    errorHandle(err, web);
    bounds := imageDecode.Bounds()
    dx := bounds.Dx()
    dy := bounds.Dy()
    row:=""
    for h := 0; h < dy; h++ {
        
        for w := 0; w < dx; w++ {
            colorRgb := imageDecode.At(w,h)
            _,g,_,_:=colorRgb.RGBA()
            g_uint8 := uint8(g >>8)
            sum := (255-g_uint8)/16
            row +=symbol[sum]
        }
        row += "\n"
        // fmt.Println(" ")
        
    }
    web.Write([]byte("圖片寬："+strconv.Itoa(dx)+" 圖片高："+strconv.Itoa(dy)+"<br>"))
    fmt.Println(dx,dy)
    elapsed := time.Since(t1)
    fmt.Println("單線程時間:" , elapsed)
    web.Write([]byte("單線程時間：" + elapsed.String()+"<br>"))
    return row
}
func imageToSymbol2(name string, web http.ResponseWriter)  string{

    t2:=time.Now()
    fmt.Println(name)
    file, err := os.Open(name)
    errorHandle(err, web);
    defer file.Close()
    imageDecode, _, err := image.Decode(file)
    if imageDecode.Bounds().Dx()>300 {
        imageDecode = resize.Resize(300,0,imageDecode,resize.Lanczos3)
    }
    errorHandle(err, web);
    bounds := imageDecode.Bounds()
    dx := bounds.Dx()
    dy := bounds.Dy()
    
    traversal1:=make (chan string )
    traversal2:=make (chan string )
    go func ()  {
        row:=""
        for h := 0; h < dy/2; h++ {
            for w := 0; w < dx; w++ {
                colorRgb := imageDecode.At(w,h)
                _,g,_,_:=colorRgb.RGBA()
                g_uint8 := uint8(g >>8)
                sum := (255-g_uint8)/16
                row +=symbol[sum]
            }
            row += "\n"
        }
        traversal1<- row
    }()
    go func ()  {
        row:=""
        for h := dy/2; h < dy; h++ {
            for w := 0; w < dx; w++ {
                colorRgb := imageDecode.At(w,h)
                _,g,_,_:=colorRgb.RGBA()
                g_uint8 := uint8(g >>8)
                sum := (255-g_uint8)/16
                row +=symbol[sum]
            }
            row += "\n"
        }
        traversal2<- row
    }()
    result:=<-traversal1+<-traversal2
    close(traversal1)
    close(traversal2)
    // fmt.Println(result)
    fmt.Println(dx,dy)
    elapsed := time.Since(t2)
    fmt.Println("多線程時間:" , elapsed)
    web.Write([]byte("多線程時間：" + elapsed.String()))
    return result
}
// 显示图片接口
func showPicHandle( w http.ResponseWriter, req *http.Request ) {
    file, err := os.Open("." + req.URL.Path)
    errorHandle(err, w);
 
    defer file.Close()
    buff, err := ioutil.ReadAll(file)
    errorHandle(err, w);
    w.Write(buff)
}
 
// 统一错误输出接口
func errorHandle(err error, w http.ResponseWriter) {
    if  err != nil {
        tmpl := template.Must(template.ParseFiles("./index.html"))
	    data:=new(IndexData)
	    data.Title = pageTitle
        data.Content =pageContent
        
        data.Error = err.Error()
        tmpl.Execute(w,data)
        fmt.Println(err)
    }
}
func main()  {
	// http.HandleFunc("/",indexPage)
	// http.HandleFunc("/index",indexPage)
	// http.HandleFunc("/front.js",frontJS)
	http.HandleFunc("/upload/",uploadHandle)
	http.HandleFunc("/uploaded/",showPicHandle)

	err:=http.ListenAndServe(":3333", nil)
	if err != nil{
		log.Fatal("ListenAndServe:",err)
	}
}
