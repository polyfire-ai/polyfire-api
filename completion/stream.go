package completion

import (
	"encoding/json"
	"net/http"

	"github.com/gorilla/websocket"
	router "github.com/julienschmidt/httprouter"
	providers "github.com/polyfact/api/llm/providers"
	utils "github.com/polyfact/api/utils"
	webrequest "github.com/polyfact/api/web_request"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin:     func(r *http.Request) bool { return true }, // For now, allow all origins

	// CheckOrigin: func(r *http.Request) bool {
	// 	allowedOrigins := []string{"http://localhost:3000"}
	// 	origin := r.Header["Origin"][0]
	// 	for _, allowedOrigin := range allowedOrigins {
	// 		if origin == allowedOrigin {
	// 			return true
	// 		}
	// 	}
	// 	return false
	// },

}

func Stream(w http.ResponseWriter, r *http.Request, _ router.Params) {
	record := r.Context().Value("recordEvent").(func(response string))
	user_id := r.Context().Value("user_id").(string)
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		utils.RespondError(w, record, "communication_error")
		return
	}
	defer conn.Close()

	messageType, p, err := conn.ReadMessage()
	if err != nil {
		utils.RespondError(w, record, "read_message_error")
		return
	}

	if messageType != websocket.TextMessage {
		utils.RespondError(w, record, "invalid_message_type")
		return
	}

	recordEventRequest := r.Context().Value("recordEventRequest").(func(request string, response string, userId string))

	record = func(response string) {
		recordEventRequest(string(p), response, user_id)
	}

	var input GenerateRequestBody

	err = json.Unmarshal(p, &input)
	if err != nil {
		utils.RespondError(w, record, "invalid_json")
		return
	}

	chan_res, err := GenerationStart(user_id, input)
	if err != nil {
		if err != nil {
			switch err {
			case webrequest.WebsiteExceedsLimit:
				utils.RespondError(w, record, "error_website_exceeds_limit")
			case webrequest.WebsitesContentExceeds:
				utils.RespondError(w, record, "error_websites_content_exceeds")
			case webrequest.NoContentFound:
				utils.RespondError(w, record, "error_no_content_found")
			case webrequest.FetchWebpageError:
				utils.RespondError(w, record, "error_fetch_webpage")
			case webrequest.ParseContentError:
				utils.RespondError(w, record, "error_parse_content")
			case webrequest.VisitBaseURLError:
				utils.RespondError(w, record, "error_visit_base_url")
			case NotFound:
				utils.RespondError(w, record, "not_found")
			case UnknownModelProvider:
				utils.RespondError(w, record, "invalid_model_provider")
			case RateLimitReached:
				utils.RespondError(w, record, "rate_limit_reached")
			default:
				utils.RespondError(w, record, "internal_error")
			}
			return
		}
	}

	result := providers.Result{
		Result:     "",
		TokenUsage: providers.TokenUsage{Input: 0, Output: 0},
	}

	chan_stop := make(chan bool)
	go func() {
		for {
			size, message, _ := conn.ReadMessage()
			if string(message) == "STOP" {
				chan_stop <- true
			}
			if size == -1 {
				break
			}
		}
	}()

	total_result := ""
generation_loop:
	for v := range *chan_res {
		result.Result += v.Result
		result.TokenUsage.Input += v.TokenUsage.Input
		result.TokenUsage.Output += v.TokenUsage.Output

		if len(v.Ressources) > 0 {
			result.Ressources = v.Ressources
		}
		select {
		case <-chan_stop:
			break generation_loop
		default:
		}

		total_result += v.Result
		if v.Result != "" {
			err = conn.WriteMessage(websocket.TextMessage, []byte(v.Result))
			if err != nil {
				utils.RespondError(w, record, "write_message_error")
				return
			}
		}
	}

	if input.MemoryId != nil && *input.MemoryId != "" && input.Infos {
		infosJSON, err := json.Marshal(result)

		infos := "[INFOS]:" + string(infosJSON)
		byteMessage := []byte(infos)

		err = conn.WriteMessage(websocket.TextMessage, byteMessage)
		if err != nil {
			utils.RespondError(w, record, "write_info_error")
			return
		}
	}

	record(total_result)

	err = conn.WriteMessage(websocket.TextMessage, []byte(""))
	if err != nil {
		utils.RespondError(w, record, "write_end_message_error")
		return
	}
}
