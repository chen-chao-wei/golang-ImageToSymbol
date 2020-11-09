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
    "encoding/base64"
)
//特殊import 只執行庫的init() 不使用其他方法
import _ "image/jpeg"
import _ "image/png"

type IndexData struct {
	Title string
    Content string
    Filename string
    Error string
    Result string
    GoroutineResult string
    //前端圖片格式
    ImageHeight string
    ImageWidth string
    ImageBase64 string
}

const pageTitle = "將上傳圖片轉成文字符號"
const pageContent ="寬度若大於300像素，會等比壓縮至300像素"
var symbol = [16]string{" ",".","-","^","~","!","(",")","=","%","&","$","+","*","#","@" } 
    
// 上傳圖片
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
        // 接收圖片
        uploadFile, handle, err := req.FormFile("image")
        fmt.Println(err)
        if err != nil{
            errorHandle(err, w)
            return
        }
        
        // 檢查圖片副檔名
        ext := strings.ToLower(path.Ext(handle.Filename))
        if ext != ".jpg" && ext != ".jpeg" && ext != ".png" {
            errorHandle(errors.New("只支援jpg/jpeg/png檔案格式"), w);
            return
        }
        
		saveFileName := []byte(handle.Filename)
        fmt.Println(string(saveFileName))
        // 保存圖片
        os.Mkdir("./uploaded/", 0777)
        saveFile, err := os.OpenFile("./uploaded/" + string(saveFileName) , os.O_WRONLY|os.O_CREATE, 0666);
        errorHandle(err, w)
        io.Copy(saveFile, uploadFile);
 
        defer uploadFile.Close()
        defer saveFile.Close()
        // 上傳圖片
        tmpl := template.Must(template.ParseFiles("./uploaded.html"))
	    data:=new(IndexData)
	    data.Title = pageTitle
        data.Content =pageContent
        data.Filename =handle.Filename
        f,_:=ioutil.ReadFile("./uploaded/"+string(saveFileName))
       
        
        data.ImageBase64 = base64.StdEncoding.EncodeToString(f)
        
        data.Result,data.ImageWidth,data.ImageHeight = imageToSymbol("./uploaded/"+string(saveFileName),w)
        data.GoroutineResult = imageToSymbol2("./uploaded/"+string(saveFileName),w)
        
        tmpl.Execute(w,data)
    }
}
//圖片轉符號（單線程）
func imageToSymbol(name string, web http.ResponseWriter)  (string ,string,string){
    t1:=time.Now()
    fmt.Println(name)
    file, err := os.Open(name)
    errorHandle(err, web);
    defer file.Close()
    imageDecode, _, err := image.Decode(file)
    //重設圖片大小
    if imageDecode.Bounds().Dx()>300 || imageDecode.Bounds().Dy()>300 {
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
            //取用符號陣列
            row +=symbol[sum]
        }
        row += "\n"
        // fmt.Println(" ")
        
    }
    
    fmt.Println(dx,dy)
    elapsed := time.Since(t1)
    fmt.Println("單線程時間:" , elapsed)
    web.Write([]byte("<p style='border-style:solid;border-color:blue;'>單線程時間：" + elapsed.String()+"</p>"))
    return row,strconv.Itoa(dx),strconv.Itoa(dy)
}
//圖片轉符號（兩線程）
func imageToSymbol2(name string, web http.ResponseWriter)  string{

    t2:=time.Now()
    fmt.Println(name)
    file, err := os.Open(name)
    errorHandle(err, web);
    defer file.Close()
    imageDecode, _, err := image.Decode(file)
    if imageDecode.Bounds().Dx()>300 || imageDecode.Bounds().Dy()>300{
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
    fmt.Println("雙線程時間:" , elapsed)
    web.Write([]byte("<p style='border-style:solid;border-color:red;'>雙線程時間：" + elapsed.String()+"</p>"))
    return result
}
// 顯示原始圖片
func showPicHandle( w http.ResponseWriter, req *http.Request ) {
    file, err := os.Open("." + req.URL.Path)
    errorHandle(err, w);
 
    defer file.Close()
    buff, err := ioutil.ReadAll(file)
    errorHandle(err, w);
    w.Write(buff)
}
 
// 統一錯誤輸出
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
	http.HandleFunc("/upload/",uploadHandle)
	http.HandleFunc("/uploaded/",showPicHandle)

	err:=http.ListenAndServe(":3333", nil)
	if err != nil{
		log.Fatal("ListenAndServe:",err)
	}
}
