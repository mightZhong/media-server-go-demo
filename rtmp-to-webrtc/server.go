package main

import (
	"encoding/json"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/gofrs/uuid"
	"github.com/gorilla/websocket"
	"math/rand"
	"net/http"
	"net/url"
	"strings"
	"time"

	mediaserver "github.com/notedit/media-server-go"
	"github.com/notedit/media-server-go-demo/rtmp-to-webrtc/rtmpstreamer"
	"github.com/notedit/sdp"

	rtmp "github.com/notedit/rtmp-lib"
	"github.com/notedit/rtmp-lib/av"
)

const (
	videoPt    = 100
	audioPt    = 96
	videoCodec = "h264"
	audioCodec = "opus"
)

type Message struct {
	Cmd string `json:"cmd,omitempty"`
	Sdp string `json:"sdp,omitempty"`
}

var endpoint = mediaserver.NewEndpoint("121.199.17.60")

var conn *websocket.Conn

var rtmpStreamer = rtmpstreamer.NewRtmpStreamer(Capabilities["audio"], Capabilities["video"])

var upGrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

var Capabilities = map[string]*sdp.Capability{
	"audio": &sdp.Capability{
		Codecs: []string{"opus"},
	},
	"video": &sdp.Capability{
		Codecs: []string{"h264"},
		Rtx:    true,
		Rtcpfbs: []*sdp.RtcpFeedback{
			&sdp.RtcpFeedback{
				ID: "goog-remb",
			},
			&sdp.RtcpFeedback{
				ID: "transport-cc",
			},
			&sdp.RtcpFeedback{
				ID:     "ccm",
				Params: []string{"fir"},
			},
			&sdp.RtcpFeedback{
				ID:     "nack",
				Params: []string{"pli"},
			},
		},
		Extensions: []string{
			"urn:3gpp:video-orientation",
			"http://www.ietf.org/id/draft-holmer-rmcat-transport-wide-cc-extensions-01",
			"http://www.webrtc.org/experiments/rtp-hdrext/abs-send-time",
		},
	},
}

func connectWebsocket() *websocket.Conn {

	u := url.URL{Scheme: "ws", Host: "121.199.18.43:8188"}
	fmt.Printf("connecting to %s\n", u.String())
	d := websocket.DefaultDialer
	d.Subprotocols = []string{"janus-protocol"}

	conn, _, err := d.Dial(u.String(), nil)
	if err != nil {
		fmt.Printf("dial: %s", err)
	} else {
		fmt.Println("Sucess")
	}

	return conn
}

func genRandomStr() string {
	rand.Seed(time.Now().UnixNano())
	chars := []rune("ABCDEFGHIJKLMNOPQRSTUVWXYZ" +
		"abcdefghijklmnopqrstuvwxyz" +
		"0123456789")
	length := 12
	var b strings.Builder
	for i := 0; i < length; i++ {
		b.WriteRune(chars[rand.Intn(len(chars))])
	}
	str := b.String()
	return str
}

type Data struct {
	Description string `json:"description,omitempty"`
	Id          int64  `json:"id"`
	Mtype       string `json:"mtype,omitempty"`
	Room        int    `json:"room,omitempty"`
	Videoroom   string `json:"videoroom,omitempty"`
}

type Body struct {
	Request     string `json:"request,omitempty"`
	Mtype       string `json:"mtype,omitempty"`
	Participant string `json:"participant,omitempty"`
	Ptype       string `json:"ptype,omitempty"`
	Room        int    `json:"room,omitempty"`
	Alinip      string `json:"alinip,omitempty"`
	Alinport    string `json:"alinport,omitempty"`
	Agent       string `json:"agent,omitempty"`
	Bitrate     int    `json:"bitrate,omitempty"`
	Display     string `json:"display,omitempty"`
	audio       bool   `json:"audio,omitempty"`
	data        bool   `json:"data,omitempty"`
	video       bool   `json:"video,omitempty"`
}

type Plugindata struct {
	Data   Data   `json:"data,omitempty"`
	Plugin string `json:"plugin,omitempty"`
}

type Jsep struct {
	Type string `json:"type,omitempty"`
	Sdp  string `json:"sdp,omitempty"`
}

type Request struct {
	Janus       string `json:"janus,omitempty"`
	Participant string `json:"participant,omitempty"`
	Plugin      string `json:"plugin,omitempty"`
	Jsep        *Jsep  `json:"jsep,omitempty"`
	Body        *Body  `json:"body,omitempty"`
	Session_id  int64  `json:"session_id,omitempty"`
	Handle_id   int64  `json:"handle_id,omitempty"`
	Transaction string `json:"transaction,omitempty"`
}

type Response struct {
	Janus       string     `json:"janus,omitempty"`
	Jsep        *Jsep      `json:"jsep,omitempty"`
	Plugindata  Plugindata `json:"plugindata,omitempty"`
	Session_id  int64      `json:"session_id,omitempty"`
	Transaction string     `json:"transaction,omitempty"`
	Sender      int64      `json:"sender,omitempty"`
	Data        Data       `json:"data,omitempty"`
}

func createSession(conn *websocket.Conn) int64 {
	var sessionId int64 = 0

	err := conn.WriteJSON(Request{
		Janus:       "create",
		Transaction: genRandomStr(),
	})

	if err != nil {
		return sessionId
	}

	conn.SetReadDeadline(time.Now().Add(30 * time.Second))
	mt, message, err := conn.ReadMessage()
	if err != nil {
		return sessionId
	}
	if mt == websocket.TextMessage {
		fmt.Println(string(message))

		var response Response
		err = json.Unmarshal(message, &response)
		if err != nil {
			fmt.Println("Unmarshal json failed")
		}

		fmt.Printf("janus %s transaction %s data %d\n", response.Janus, response.Transaction, response.Data.Id)
		sessionId = response.Data.Id
	}
	return sessionId
}

func attachPlugin(conn *websocket.Conn, sessionId int64) int64 {
	var handlerId int64 = 0
	err := conn.WriteJSON(Request{
		Janus:       "attach",
		Participant: "QFC2hIbWwSCfVn5vg24iCA",
		Plugin:      "janus.plugin.videoroom",
		Session_id:  sessionId,
		Transaction: genRandomStr(),
	})

	if err != nil {
		return handlerId
	}
	mt, message, err := conn.ReadMessage()
	if err != nil {
		return handlerId
	}
	if mt == websocket.TextMessage {
		fmt.Println(string(message))

		var response Response
		err = json.Unmarshal(message, &response)
		if err != nil {
			fmt.Println("Unmarshal json failed")
		}

		fmt.Printf("janus %s transaction %s data %d\n", response.Janus, response.Transaction, response.Data.Id)
		handlerId = response.Data.Id
	}
	return handlerId
}

func joinRoom(conn *websocket.Conn, sessionId int64, handleId int64) bool {
	err := conn.WriteJSON(Request{
		Janus: "message",
		Body: &Body{
			Request:     "joinandconfigure",
			Mtype:       "main",
			Participant: "oRTSk5Y9yVYuEMOoEIrdyA",
			Ptype:       "publisher",
			Room:        54753,
			Alinip:      "121.40.86.32",
			Alinport:    "-1",
			Agent:       "rtmp-to-webrtc server",
			Bitrate:     4194304,
			Display:     "oRTSk5Y9yVYuEMOoEIrdyA",
		},
		Session_id:  sessionId,
		Handle_id:   handleId,
		Transaction: genRandomStr(),
	})

	if err != nil {
		return false
	}
	// read response
	mt, message, err := conn.ReadMessage()
	if err != nil {
		return false
	}
	if mt == websocket.TextMessage {
		fmt.Println(string(message))

		var response Response
		err = json.Unmarshal(message, &response)
		if err != nil {
			fmt.Println("Unmarshal json failed")
		}

		fmt.Printf("janus %s transaction %s \n", response.Janus, response.Transaction)
	}

	// read event
	mt, message, err = conn.ReadMessage()
	if err != nil {
		return false
	}
	if mt == websocket.TextMessage {
		fmt.Println(string(message))

		var response Response
		err = json.Unmarshal(message, &response)
		if err != nil {
			fmt.Println("Unmarshal json failed")
		}
		if response.Janus == "event" {
			if response.Plugindata.Data.Videoroom == "joined" {
				return true
			}
		}
	}
	return false

}

func sendOffer(conn *websocket.Conn, sessionId int64, handleId int64, sdp string) (string, error) {

	err := conn.WriteJSON(Request{
		Janus: "message",
		Body: &Body{
			Request: "configure",
			audio:   true,
			data:    false,
			video:   true,
		},
		Jsep: &Jsep{
			Type: "offer",
			Sdp:  sdp,
		},
		Session_id:  sessionId,
		Handle_id:   handleId,
		Transaction: genRandomStr(),
	})

	if err != nil {
		return "", err
	}
	// read response
	mt, message, err := conn.ReadMessage()
	if err != nil {
		return "", err
	}
	if mt == websocket.TextMessage {
		fmt.Println(string(message))

		var response Response
		err = json.Unmarshal(message, &response)
		if err != nil {
			fmt.Println("Unmarshal json failed")
		}

		fmt.Printf("janus %s transaction %s \n", response.Janus, response.Transaction)
	}

	// read event
	mt, message, err = conn.ReadMessage()
	if err != nil {
		return "", err
	}
	if mt == websocket.TextMessage {
		fmt.Println(string(message))

		var response Response
		err = json.Unmarshal(message, &response)
		if err != nil {
			fmt.Println("Unmarshal json failed")
		}
		if response.Janus == "event" {
			if response.Jsep.Type == "answer" {
				if err != nil {
					panic(err)
				}
				return response.Jsep.Sdp, nil
			}
		}
	}
	return "", err

}

func connectJanus() {
	var transport *mediaserver.Transport
	conn = connectWebsocket()
	sessionId := createSession(conn)
	handleId := attachPlugin(conn, sessionId)
	if joinRoom(conn, sessionId, handleId) {
		offer := endpoint.CreateOffer(Capabilities["video"], Capabilities["audio"])
		sdpStr := offer.String()
		println("Create offer: ")
		println(sdpStr)
		offer, err := sdp.Parse(sdpStr)
		if err != nil {
			panic(err)
		}

		asnwerStr, err := sendOffer(conn, sessionId, handleId, sdpStr)
		if err != nil {
			panic(err)
		}
		answer, err := sdp.Parse(asnwerStr)

		fmt.Printf("CreateTransport...")
		transport = endpoint.CreateTransport(answer, offer)
		fmt.Printf("SetLocalProperties...")
		transport.SetLocalProperties(offer.GetMedia("audio"), offer.GetMedia("video"))
		fmt.Printf("SetRemoteProperties...")
		transport.SetRemoteProperties(answer.GetMedia("audio"), answer.GetMedia("video"))

		outgoingStream := transport.CreateOutgoingStreamWithID(uuid.Must(uuid.NewV4()).String(), true, true)

		outgoingStream.GetVideoTracks()[0].AttachTo(rtmpStreamer.GetVideoTrack())
		outgoingStream.GetAudioTracks()[0].AttachTo(rtmpStreamer.GetAuidoTrack())
		fmt.Printf("Sleeping...")
		time.Sleep(600 * time.Second)
	}
}

func index(c *gin.Context) {

	fmt.Println("helloworld")
	c.HTML(http.StatusOK, "index.html", gin.H{})
}

func connect(c *gin.Context) {
	fmt.Printf("Starting...")
	connectJanus()
	fmt.Printf("Complete...")

}

func channel(c *gin.Context) {

	ws, err := upGrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		return
	}
	defer ws.Close()

	var transport *mediaserver.Transport

	for {
		var msg Message
		err = ws.ReadJSON(&msg)

		if err != nil {
			fmt.Println("error: ", err)
			break
		}

		if msg.Cmd == "offer" {

			offer, err := sdp.Parse(msg.Sdp)
			if err != nil {
				panic(err)
			}
			transport = endpoint.CreateTransport(offer, nil)
			transport.SetRemoteProperties(offer.GetMedia("audio"), offer.GetMedia("video"))

			answer := offer.Answer(transport.GetLocalICEInfo(),
				transport.GetLocalDTLSInfo(),
				endpoint.GetLocalCandidates(),
				Capabilities)

			transport.SetLocalProperties(answer.GetMedia("audio"), answer.GetMedia("video"))

			outgoingStream := transport.CreateOutgoingStreamWithID(uuid.Must(uuid.NewV4()).String(), true, true)

			outgoingStream.GetVideoTracks()[0].AttachTo(rtmpStreamer.GetVideoTrack())
			outgoingStream.GetAudioTracks()[0].AttachTo(rtmpStreamer.GetAuidoTrack())

			info := outgoingStream.GetStreamInfo()
			answer.AddStream(info)

			ws.WriteJSON(Message{
				Cmd: "answer",
				Sdp: answer.String(),
			})
		}

	}

}

func main() {
	server := &rtmp.Server{}

	server.HandlePublish = func(conn *rtmp.Conn) {

		var streams []av.CodecData
		var err error

		if streams, err = conn.Streams(); err != nil {
			fmt.Println(err)
			return
		}

		if err = rtmpStreamer.WriteHeader(streams); err != nil {
			fmt.Println(err)
			return
		}

		for {
			packet, err := conn.ReadPacket()
			if err != nil {
				fmt.Println(err)
				break
			}
			rtmpStreamer.WritePacket(packet)
		}
	}

	go server.ListenAndServe()

	address := ":8000"
	r := gin.Default()

	r.LoadHTMLFiles("./index.html")
	r.GET("/channel", channel)
	r.GET("/", index)
	r.GET("/connect", connect)

	r.Run(address)

}
