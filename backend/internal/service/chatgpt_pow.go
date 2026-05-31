package service

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"math/rand"
)

// PoW (Proof of Work) implementation matching chatgpt2api's pow.py
// This generates the requirements token needed for ChatGPT API authentication

var navigatorKeys = []string{
	"registerProtocolHandler−function registerProtocolHandler() { [native code] }",
	"storage−[object StorageManager]",
	"locks−[object LockManager]",
	"appCodeName−Mozilla",
	"permissions−[object Permissions]",
	"share−function share() { [native code] }",
	"webdriver−false",
	"managed−[object NavigatorManagedData]",
	"canShare−function canShare() { [native code] }",
	"vendor−Google Inc.",
	"mediaDevices−[object MediaDevices]",
	"vibrate−function vibrate() { [native code] }",
	"storageBuckets−[object StorageBucketManager]",
	"mediaCapabilities−[object MediaCapabilities]",
	"cookieEnabled−true",
	"virtualKeyboard−[object VirtualKeyboard]",
	"product−Gecko",
	"presentation−[object Presentation]",
	"onLine−true",
	"mimeTypes−[object MimeTypeArray]",
	"credentials−[object CredentialsContainer]",
	"serviceWorker−[object ServiceWorkerContainer]",
	"keyboard−[object Keyboard]",
	"gpu−[object GPU]",
	"doNotTrack",
	"serial−[object Serial]",
	"pdfViewerEnabled−true",
	"language−zh-CN",
	"geolocation−[object Geolocation]",
	"userAgentData−[object NavigatorUAData]",
	"getUserMedia−function getUserMedia() { [native code] }",
	"sendBeacon−function sendBeacon() { [native code] }",
	"hardwareConcurrency−32",
	"windowControlsOverlay−[object WindowControlsOverlay]",
}

var screenKeys = []string{
	"availWidth−2560",
	"availHeight−1400",
	"width−2560",
	"height−1440",
	"colorDepth−24",
	"pixelDepth−24",
	"availLeft−0",
	"availTop−40",
	"orientation−[object ScreenOrientation]",
	"onchange",
	"isExtended−false",
}

// BuildPoWConfig creates the PoW configuration array
func BuildPoWConfig(userAgent string) []interface{} {
	navigatorKey := navigatorKeys[rand.Intn(len(navigatorKeys))]
	screenKey := screenKeys[rand.Intn(len(screenKeys))]
	
	config := []interface{}{
		navigatorKey,
		screenKey,
		0,
		nil,
		userAgent,
		"https://chatgpt.com/",
		"zh-CN",
		"zh-CN,zh;q=0.9,en;q=0.8,en-US;q=0.7",
		0,
	}
	
	return config
}

// GeneratePoW generates a proof-of-work solution
func GeneratePoW(seed, difficulty string, config []interface{}) (string, bool) {
	// Simplified PoW - in production, this should match chatgpt2api's exact algorithm
	// For now, we'll use a fallback approach
	configJSON, _ := json.Marshal(config)
	data := fmt.Sprintf("%s%s%s", seed, difficulty, string(configJSON))
	hash := sha256.Sum256([]byte(data))
	_ = base64.StdEncoding.EncodeToString(hash[:])
	
	// Fallback format matching chatgpt2api
	fallback := "wQ8Lk5FbGpA2NcR9dShT6gYjU7VxZ4D" + base64.StdEncoding.EncodeToString([]byte(fmt.Sprintf(`"%s"`, seed)))
	
	return fallback, false
}

// BuildLegacyRequirementsToken creates the requirements token for chat-requirements endpoint
func BuildLegacyRequirementsToken(userAgent string) string {
	seed := fmt.Sprintf("%.16f", rand.Float64())
	config := BuildPoWConfig(userAgent)
	answer, _ := GeneratePoW(seed, "0fffff", config)
	return "gAAAAAC" + answer
}

// ParsePoWResources extracts PoW script sources from bootstrap response
// For now, returns empty as we're using fallback PoW
func ParsePoWResources(bootstrapHTML string) ([]string, string) {
	// In production, parse the actual script sources from HTML
	// For now, return empty to use fallback
	return []string{}, ""
}
