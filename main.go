package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"io/ioutil"
	"encoding/json"
	"strings"
	"github.com/line/line-bot-sdk-go/linebot"
	"github.com/go-redis/redis"
	"strconv"
	"math"
	"regexp"
)

type device struct {
	// Gps_num float32 `json:"gps_num"`
	// App string `json:"app"`
	// Gps_alt float32 `json:"gps_alt"`
	// Fmt_opt int `json:"fmt_opt"`
	// Device string `json:"device"`
	// S_d2 float32 `json:"s_d2"`
	S_d0 float32 `json:"s_d0"`
	// S_d1 float32 `json:"s_d1"`
	S_h0 float32 `json:"s_h0"`
	SiteName string `json:"SiteName"`
	// Gps_fix float32 `json:"gps_fix"`
	// Ver_app string `json:"ver_app"`
	Gps_lat float64 `json:"gps_lat"`
	S_t0 float32 `json:"s_t0"`
	Timestamp string `json:"timestamp"`
	Gps_lon float64 `json:"gps_lon"`
	// Date string `json:"date"`
	// Tick float32 `json:"tick"`
	Device_id string `json:"device_id"`
	// S_1 float32 `json:"s_1"`
	// S_0 float32 `json:"s_0"`
	// S_3 float32 `json:"s_3"`
	// S_2 float32 `json:"s_2"`
	// Ver_format string `json:"ver_format"`
	// Time string `json:"time"`
}

type airbox struct {
	Source string `json:"source"`
	Feeds []device `json:"feeds"`
	Version string `json:"version"`
	Num_of_records int `json:"num_of_records"`
}

type subscribeid struct {
	Device_id []string `json:"device_id"`
	Sitename []string `json:"sitename"`
}

var bot *linebot.Client
var airbox_json airbox
var lass_json airbox
var maps_json airbox
var all_device []device
var history_json subscribeid
var	client=redis.NewClient(&redis.Options{
		Addr:"hipposerver.ddns.net:6379",
		Password:"",
		DB:0,
	})

func main() {
	url := "https://data.lass-net.org/data/last-all-airbox.json"
	req, _ := http.NewRequest("GET", url, nil)
	res, _ := http.DefaultClient.Do(req)
	defer res.Body.Close()
	body, _ := ioutil.ReadAll(res.Body)
	errs := json.Unmarshal(body, &airbox_json)
	if errs != nil {
		fmt.Println(errs)
	}

	url = "https://data.lass-net.org/data/last-all-lass.json"
	req, _ = http.NewRequest("GET", url, nil)
	res, _ = http.DefaultClient.Do(req)
	defer res.Body.Close()
	body, _ = ioutil.ReadAll(res.Body)
	errs = json.Unmarshal(body, &lass_json)
	if errs != nil {
		fmt.Println(errs)
	}

	url = "https://data.lass-net.org/data/last-all-maps.json"
	req, _ = http.NewRequest("GET", url, nil)
	res, _ = http.DefaultClient.Do(req)
	defer res.Body.Close()
	body, _ = ioutil.ReadAll(res.Body)
	errs = json.Unmarshal(body, &maps_json)
	if errs != nil {
		fmt.Println(errs)
	}

	all_device=append(maps_json.Feeds,lass_json.Feeds...)
	all_device=append(all_device,airbox_json.Feeds...)

	url = "https://data.lass-net.org/data/airbox_list.json"
	req, _ = http.NewRequest("GET", url, nil)
	res, _ = http.DefaultClient.Do(req)
	defer res.Body.Close()
	body, _ = ioutil.ReadAll(res.Body)
	errs = json.Unmarshal(body, &history_json)
	if errs != nil {
		fmt.Println(errs)
	}
	// pushmessage()
	// fmt.Println(airbox_json)
	var err error
	bot, err = linebot.New(os.Getenv("ChannelSecret"), os.Getenv("ChannelAccessToken"))
	// _,_=bot.PushMessage("U3617adbdd46283d7e859f36302f4f471", linebot.NewTextMessage("hi!")).Do()
	log.Println("Bot:", bot, " err:", err)
	http.HandleFunc("/callback", callbackHandler)
	port := os.Getenv("PORT")
	addr := fmt.Sprintf(":%s", port)
	http.ListenAndServe(addr, nil)

}

func callbackHandler(w http.ResponseWriter, r *http.Request) {
	events, err := bot.ParseRequest(r)

	if err != nil {
		if err == linebot.ErrInvalidSignature {
			w.WriteHeader(400)
		} else {
			w.WriteHeader(500)
		}
		return
	}

	for _, event := range events {
		if event.Type == linebot.EventTypeMessage {
			switch message := event.Message.(type) {
			case *linebot.TextMessage:
				var txtmessage string
				inText := strings.ToLower(message.Text)
				if strings.Contains(inText,"訂閱")||strings.Contains(inText,"@"){
					userID:=event.Source.UserID
					// pong, _ := client.Ping().Result()
					// txtmessage=pong
					for i:=0; i<len(history_json.Device_id); i++ {
						if strings.Contains(inText,strings.ToLower(history_json.Device_id[i]))||strings.Contains(inText,strings.ToLower(history_json.Sitename[i])) {
							val, err:=client.Get(history_json.Device_id[i]).Result()
							if err!=nil{
								if strings.Contains(inText,"取消")||strings.Contains(inText,"-c"){
									txtmessage="你並沒有訂閱此ID。"
									break
								}
								client.Set(history_json.Device_id[i],userID,0)
								txtmessage="訂閱成功!"
								break
							}
							if strings.Contains(inText,"取消")||strings.Contains(inText,"-c"){
								stringSlice:=strings.Split(val,",")
								if stringInSlice(userID,stringSlice){
									if len(stringSlice)==1{
										err=client.Del(history_json.Device_id[i]).Err()
										if err!=nil{
											txtmessage=err.Error()
											break
										}
										txtmessage="取消訂閱成功!"
										break
									}else{
										var s []string
										s = removeStringInSlice(stringSlice, userID)
										var afterremoved string
										for k:=0; k<len(s); k++{
											if k==0{
												afterremoved=s[k]
												continue
											}
											afterremoved=afterremoved+","+s[k]
										}
										client.Set(history_json.Device_id[i],afterremoved,0)
										// fmt.Println(s)
										// fmt.Println(err)
										txtmessage="取消訂閱成功!"
										break
									}
								}else{
									txtmessage="你並沒有訂閱此ID。"
									break
								}
							}
							stringSlice:=strings.Split(val,",")
							if stringInSlice(userID,stringSlice){
								txtmessage="您已訂閱過此ID!"
								break
							} else{
								val=val+","+userID
								client.Set(history_json.Device_id[i],val,0)
								txtmessage="訂閱成功!"
								break
							}
						}
					}
				} else if strings.Contains(inText,"門檻值")||strings.Contains(inText,"-t"){
					// 新增門檻值
					userID:=event.Source.UserID
					client2:=redis.NewClient(&redis.Options{
							Addr:"hipposerver.ddns.net:6379",
							Password:"",
							DB:1,
					})
					re:=regexp.MustCompile("[0-9]+")
					number:=re.FindAllString(inText,-1)
					threshold:=number[len(number)-1]
					val, err:=client2.Get(userID).Result()
					if err!=nil{
						client2.Set(userID,threshold,0)
						txtmessage="已為您將門檻值設為"+threshold+"，當您訂閱的AirBox超過這個門檻值將會發出警告！"
					} else{
						client2.Set(userID,threshold,0)
						txtmessage="已為您將門檻值從"+val+"改為"+threshold+"，當您訂閱的AirBox超過這個門檻值將會發出警告！"
					}
				} else if strings.Contains(inText,"help"){
					txtmessage="[HELP]\n"
					txtmessage=txtmessage+"1. 訂閱機器：@Device_id/SiteName, eg. @28C2DDDD47A8(id大小寫不拘) 或 @台北市龍安國小\n"
					txtmessage=txtmessage+"2. 取消訂閱：-c @Device_id/SiteName, eg. -c @28C2DDDD47A8(id大小寫不拘) 或 -c @台北市龍安國小\n"
					txtmessage=txtmessage+"3. 門檻值：-t 門檻值, eg. -t 100\n"
					txtmessage=txtmessage+"4. 地點查詢：點選左下角'+'號，點選'傳送位置訊息'，分享位置即可\n"
					txtmessage=txtmessage+"5. 查詢已訂閱列表：輸入'-l'"
				} else if strings.Contains(inText,"-l"){
					userID:=event.Source.UserID
					subscribbed_device:=client.Keys("*").Val()
					var list []string
					for i:=0; i<len(subscribbed_device); i++ {
						val,_:=client.Get(subscribbed_device[i]).Result()
						stringSlice:=strings.Split(val,",")
						if stringInSlice(userID,stringSlice){
							list=append(list,subscribbed_device[i])
						}
					}
					txtmessage="以下為您已訂閱之設備：\n"
					for j:=0; j<len(list); j++{
						txtmessage=txtmessage+list[j]+"\n"
					}
				} else{
					for i:=0; i<len(all_device); i++ {
						if strings.Contains(inText,strings.ToLower(all_device[i].Device_id)) {
							txtmessage="Device_id: "+all_device[i].Device_id+"\n"
							txtmessage=txtmessage+"Site Name: "+all_device[i].SiteName+"\n"
							txtmessage=txtmessage+"Location: ("+strconv.FormatFloat(float64(all_device[i].Gps_lon),'f',3,64)+","+strconv.FormatFloat(float64(all_device[i].Gps_lat),'f',3,64)+")"+"\n"
							txtmessage=txtmessage+"Timestamp: "+all_device[i].Timestamp+"\n"
							txtmessage=txtmessage+"PM2.5: "+strconv.FormatFloat(float64(all_device[i].S_d0),'f',0,64)+"\n"
							txtmessage=txtmessage+"Humidity: "+strconv.FormatFloat(float64(all_device[i].S_h0),'f',0,64)+"\n"
							txtmessage=txtmessage+"Temperature: "+strconv.FormatFloat(float64(all_device[i].S_t0),'f',0,64)
							break
						} else if len(all_device[i].SiteName)!=0 && strings.Contains(inText,strings.ToLower(all_device[i].SiteName)){
							txtmessage="Device_id: "+all_device[i].Device_id+"\n"
							txtmessage=txtmessage+"Site Name: "+all_device[i].SiteName+"\n"
							txtmessage=txtmessage+"Location: ("+strconv.FormatFloat(float64(all_device[i].Gps_lon),'f',3,64)+","+strconv.FormatFloat(float64(all_device[i].Gps_lat),'f',3,64)+")"+"\n"
							txtmessage=txtmessage+"Timestamp: "+all_device[i].Timestamp+"\n"
							txtmessage=txtmessage+"PM2.5: "+strconv.FormatFloat(float64(all_device[i].S_d0),'f',0,64)+"\n"
							txtmessage=txtmessage+"Humidity: "+strconv.FormatFloat(float64(all_device[i].S_h0),'f',0,64)+"\n"
							txtmessage=txtmessage+"Temperature: "+strconv.FormatFloat(float64(all_device[i].S_t0),'f',0,64)
							break
						}
					}
				}
				if len(txtmessage)==0{
					txtmessage="很抱歉! 這個AirBox ID不存在或不提供即時資訊查詢，或指令錯誤，如需要查詢指令表請在輸入框中輸入'help'。"
				}
				if _, err = bot.ReplyMessage(event.ReplyToken, linebot.NewTextMessage(txtmessage)).Do(); err != nil {
					log.Print(err)
				}
			case *linebot.LocationMessage:
				var txtmessage string
				lat := message.Latitude
				lon := message.Longitude
				minD := math.MaxFloat64
				var reply_lat float64
				var reply_lon float64
				var reply_add string
				for i:=0; i<len(all_device); i++ {
					if all_device[i].Gps_lat<=lat+0.05 && all_device[i].Gps_lat>=lat-0.05 && all_device[i].Gps_lon<=lon+0.05 && all_device[i].Gps_lon>=lon-0.05{
						D:=distanceInKmBetweenEarthCoordinates(lat, lon, all_device[i].Gps_lat, all_device[i].Gps_lon)
						if D<minD{
							minD=D
							reply_lat=all_device[i].Gps_lat
							reply_lon=all_device[i].Gps_lon
							reply_add=all_device[i].Device_id
							txtmessage="離您所提供的位置最近的AirBox資訊如以下所示：\n"
							txtmessage=txtmessage+"Device_id: "+all_device[i].Device_id+"\n"
							txtmessage=txtmessage+"Site Name: "+all_device[i].SiteName+"\n"
							txtmessage=txtmessage+"Location: ("+strconv.FormatFloat(float64(all_device[i].Gps_lon),'f',3,64)+","+strconv.FormatFloat(float64(all_device[i].Gps_lat),'f',3,64)+")"+"\n"
							txtmessage=txtmessage+"Timestamp: "+all_device[i].Timestamp+"\n"
							txtmessage=txtmessage+"PM2.5: "+strconv.FormatFloat(float64(all_device[i].S_d0),'f',0,64)+"\n"
							txtmessage=txtmessage+"Humidity: "+strconv.FormatFloat(float64(all_device[i].S_h0),'f',0,64)+"\n"
							txtmessage=txtmessage+"Temperature: "+strconv.FormatFloat(float64(all_device[i].S_t0),'f',0,64)
						}
					} else{
						continue
					}
				}
				if len(txtmessage)==0{
					txtmessage="很抱歉這附近1公里內沒有任何上線的AirBox。"
					if _, err = bot.ReplyMessage(event.ReplyToken, linebot.NewTextMessage(txtmessage)).Do(); err != nil {
						log.Print(err)
					}
				} else{
					if _, err = bot.ReplyMessage(event.ReplyToken, linebot.NewTextMessage(txtmessage), linebot.NewLocationMessage("Device Location",reply_add,reply_lat,reply_lon)).Do(); err != nil {
						log.Print(err)
					}
				}
			}
		}
	}
}

func stringInSlice(a string, list []string) bool {
    for _, b := range list {
        if b == a {
            return true
        }
    }
    return false
}

func removeStringInSlice(s []string, r string) []string {
    for i, v := range s {
        if v == r {
            return append(s[:i], s[i+1:]...)
        }
    }
    return s
}

func degreesToRadians(degrees float64) float64 {
  return degrees * math.Pi / 180;
}

func distanceInKmBetweenEarthCoordinates(lat1 float64, lon1 float64, lat2 float64, lon2 float64) float64 {
  var earthRadiusKm = 6371.0;

  var dLat = degreesToRadians(lat2-lat1);
  var dLon = degreesToRadians(lon2-lon1);

  lat1 = degreesToRadians(lat1);
  lat2 = degreesToRadians(lat2);

  var a = math.Sin(dLat/2) * math.Sin(dLat/2) +
          math.Sin(dLon/2) * math.Sin(dLon/2) * math.Cos(lat1) * math.Cos(lat2); 
  var c = 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a)); 
  return earthRadiusKm * c;
}

