package youtube

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"regexp"
	"strings"
)

var (
	ytplayerConfigRE = regexp.MustCompile(`;ytplayer\.config\s*=\s*({.*?});`)
	errNotYoutube    = fmt.Errorf("youtube: not a youtube video")
)

type Youtube struct {
	ID string
	// Formats is a map of format IDs to URLs.
	Formats map[string]string
}

func (y *Youtube) URL() *url.URL {
	return &url.URL{
		Scheme:   "https",
		Host:     "www.youtube.com",
		Path:     "/watch",
		RawQuery: url.Values{"v": {y.ID}}.Encode(),
	}
}

func New(id string) (*Youtube, error) {
	y := &Youtube{
		ID: id,
	}
	resp, err := http.Get(y.URL().String())
	if err != nil {
		return nil, err
	}
	b, err := ioutil.ReadAll(resp.Body)
	resp.Body.Close()
	if err != nil {
		return nil, err
	}
	matches := ytplayerConfigRE.FindSubmatch(b)
	if matches == nil {
		return nil, errNotYoutube
	}
	var c youtubeConfig
	if err := json.Unmarshal(matches[1], &c); err != nil {
		return nil, err
	}
	info := c.Args
	if info.UrlEncodedFmtStreamMap == "" {
		return nil, fmt.Errorf("youtube: no stream_map present")
	}
	if info.Token == "" {
		return nil, fmt.Errorf("youtube: no token parameter")
	}
	encoded_url_map := info.UrlEncodedFmtStreamMap + "," + info.AdaptiveFmts
	y.Formats = make(map[string]string)
	for _, s := range strings.Split(encoded_url_map, ",") {
		v, err := url.ParseQuery(s)
		if err != nil {
			continue
		}
		if itag, url := v["itag"], v["url"]; len(itag) == 0 || len(url) == 0 {
			continue
		}
		format_id := v["itag"][0]
		u := v["url"][0]
		y.Formats[format_id] = u
	}
	return y, nil
}

type videoInfo struct {
	AccountPlaybackToken            string `json:"account_playback_token"`
	Ad3Module                       string `json:"ad3_module"`
	AdDevice                        string `json:"ad_device"`
	AdFlags                         string `json:"ad_flags"`
	AdLoggingFlag                   string `json:"ad_logging_flag"`
	AdPreroll                       string `json:"ad_preroll"`
	AdSlots                         string `json:"ad_slots"`
	AdTag                           string `json:"ad_tag"`
	AdaptiveFmts                    string `json:"adaptive_fmts"`
	AdsenseVideoDocID               string `json:"adsense_video_doc_id"`
	Aftv                            bool   `json:"aftv"`
	Afv                             bool   `json:"afv"`
	AfvAdTag                        string `json:"afv_ad_tag"`
	AfvAdTagRestrictedToInstream    string `json:"afv_ad_tag_restricted_to_instream"`
	AfvInstreamMax                  string `json:"afv_instream_max"`
	AfvInvideoAdTag                 string `json:"afv_invideo_ad_tag"`
	AllowEmbed                      string `json:"allow_embed"`
	AllowHtml5Ads                   string `json:"allow_html5_ads"`
	AllowRatings                    string `json:"allow_ratings"`
	ApplyFadeOnMidrolls             bool   `json:"apply_fade_on_midrolls"`
	AsLaunchedInCountry             string `json:"as_launched_in_country"`
	Atc                             string `json:"atc"`
	Author                          string `json:"author"`
	AvgRating                       string `json:"avg_rating"`
	C                               string `json:"c"`
	CafeExperimentID                string `json:"cafe_experiment_id"`
	Cid                             string `json:"cid"`
	Cl                              string `json:"cl"`
	ClipboardSwf                    string `json:"clipboard_swf"`
	CoreDbp                         string `json:"core_dbp"`
	Cr                              string `json:"cr"`
	CsiPageType                     string `json:"csi_page_type"`
	Dashmpd                         string `json:"dashmpd"`
	Dbp                             string `json:"dbp"`
	Dclk                            bool   `json:"dclk"`
	DynamicAllocationAdTag          string `json:"dynamic_allocation_ad_tag"`
	Enablecsi                       string `json:"enablecsi"`
	Enablejsapi                     int    `json:"enablejsapi"`
	Eventid                         string `json:"eventid"`
	ExcludedAds                     string `json:"excluded_ads"`
	FadeInDurationMilliseconds      string `json:"fade_in_duration_milliseconds"`
	FadeInStartMilliseconds         string `json:"fade_in_start_milliseconds"`
	FadeOutDurationMilliseconds     string `json:"fade_out_duration_milliseconds"`
	FadeOutStartMilliseconds        string `json:"fade_out_start_milliseconds"`
	Fexp                            string `json:"fexp"`
	FmtList                         string `json:"fmt_list"`
	GutTag                          string `json:"gut_tag"`
	Hl                              string `json:"hl"`
	HostLanguage                    string `json:"host_language"`
	Idpj                            string `json:"idpj"`
	Instream                        bool   `json:"instream"`
	InstreamLong                    bool   `json:"instream_long"`
	Invideo                         bool   `json:"invideo"`
	Iurl                            string `json:"iurl"`
	Iurlhq                          string `json:"iurlhq"`
	Iurlmaxres                      string `json:"iurlmaxres"`
	Iurlmq                          string `json:"iurlmq"`
	Iurlsd                          string `json:"iurlsd"`
	Iv3Module                       string `json:"iv3_module"`
	IvInvideoURL                    string `json:"iv_invideo_url"`
	IvLoadPolicy                    string `json:"iv_load_policy"`
	IvModule                        string `json:"iv_module"`
	Keywords                        string `json:"keywords"`
	Ldpj                            string `json:"ldpj"`
	LengthSeconds                   string `json:"length_seconds"`
	LoaderUrl                       string `json:"loaderUrl"`
	Loeid                           string `json:"loeid"`
	Loudness                        string `json:"loudness"`
	MaxDynamicAllocationAdTagLength string `json:"max_dynamic_allocation_ad_tag_length"`
	MidrollFreqcap                  string `json:"midroll_freqcap"`
	MidrollPrefetchSize             string `json:"midroll_prefetch_size"`
	Mpu                             bool   `json:"mpu"`
	Mpvid                           string `json:"mpvid"`
	NoGetVideoLog                   string `json:"no_get_video_log"`
	Of                              string `json:"of"`
	Oid                             string `json:"oid"`
	Plid                            string `json:"plid"`
	Pltype                          string `json:"pltype"`
	ProbeURL                        string `json:"probe_url"`
	Ptk                             string `json:"ptk"`
	PyvAdChannel                    string `json:"pyv_ad_channel"`
	PyvInRelatedCafeExperimentID    string `json:"pyv_in_related_cafe_experiment_id"`
	Sffb                            bool   `json:"sffb"`
	Shortform                       bool   `json:"shortform"`
	ShowContentThumbnail            bool   `json:"show_content_thumbnail"`
	ShowPyvInRelated                bool   `json:"show_pyv_in_related"`
	Ssl                             string `json:"ssl"`
	StoryboardSpec                  string `json:"storyboard_spec"`
	T                               string `json:"t"`
	TagForChildDirected             bool   `json:"tag_for_child_directed"`
	ThumbnailURL                    string `json:"thumbnail_url"`
	Timestamp                       string `json:"timestamp"`
	Title                           string `json:"title"`
	Tmi                             string `json:"tmi"`
	Token                           string `json:"token"`
	Trueview                        bool   `json:"trueview"`
	Ucid                            string `json:"ucid"`
	UrlEncodedFmtStreamMap          string `json:"url_encoded_fmt_stream_map"`
	VideoID                         string `json:"video_id"`
	VideostatsPlaybackBaseURL       string `json:"videostats_playback_base_url"`
	ViewCount                       string `json:"view_count"`
	Vm                              string `json:"vm"`
	Watermark                       string `json:"watermark"`
}

type youtubeConfig struct {
	Args   videoInfo `json:"args"`
	Assets struct {
		Css string `json:"css"`
		Js  string `json:"js"`
	} `json:"assets"`
	Attrs struct {
		ID string `json:"id"`
	} `json:"attrs"`
	Html5    bool `json:"html5"`
	Messages struct {
		PlayerFallback []string `json:"player_fallback"`
	} `json:"messages"`
	MinVersion string `json:"min_version"`
	Params     struct {
		Allowfullscreen   string `json:"allowfullscreen"`
		Allowscriptaccess string `json:"allowscriptaccess"`
		Bgcolor           string `json:"bgcolor"`
	} `json:"params"`
	Sts      int    `json:"sts"`
	URL      string `json:"url"`
	UrlV8    string `json:"url_v8"`
	UrlV9as2 string `json:"url_v9as2"`
}
