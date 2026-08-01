package ops

import (
	"regexp"
	"strings"
)

// Detection tables for Parse User Agent. Each table is ordered: specific
// products come before the platform browsers and generic engines they are
// built on, and the first matching rule decides the category. The rules were
// derived from the structure of real user-agent strings and from the observed
// input/output behaviour of CyberChef's parser over a large corpus, so common
// agents parse identically.

// uaBrowserRules identify the browser name and version.
var uaBrowserRules = []uaRule{
	uaNamed("Mobile Chrome", `crios/([\w.]+)`),
	uaNamed("Edge", `edg(?:e|ios|a)?/([\w.]+)`),
	uaCapNamed(`(opera mini)/([-\w.]+)`),
	uaNamed("Opera Mini", `opios/([\w.]+)`),
	uaNamed("Opera GX", `\bopx/([\w.]+)`),
	uaNamed("Opera", `\bopr/([\w.]+)`),
	uaCapNamed(`(opera)(?:.+version/|[/ ]+)([\w.]+)`),
	uaNamed("Opera Neon", ` mms/([\w.]+)`),
	uaNamed("Opera Touch", ` opt/([\w.]+)`),
	uaNamed("Opera Coast", `coast/([\w.]+)`),
	uaNamed("IEMobile", `iemobile[/ ]([\w.]+)`),
	uaCapNamed(`(?:ms|\()(ie) ([\w.]+)`),
	uaNamed("IE", `trident.+rv[: ]([\w.]{1,9})\b.+like gecko`),
	uaNamed("Yandex", `yabrowser/([\w.]+)`),
	// Avast and AVG ship rebadged builds of the same browser, named by
	// suffixing the brand.
	{re: regexp.MustCompile(`(?i)(avast|avg)/([\w.]+)`), apply: func(_ string, m []string, res map[string]string) bool {
		res["name"] = m[1] + " Secure Browser"
		uaSetVersion(res, m[2])
		return true
	}},
	uaNamed("Firefox Focus", `\bfocus/([\w.]+)`),
	uaNamed("DuckDuckGo", `\bddg/([\w.]+)`),
	uaNamed("DuckDuckGo", `duckduckgo/([\w.]+)`),
	uaNamed("Coc Coc", `coc_coc\w+/([\w.]+)`),
	uaNamed("Dolphin", `dolfin/([\w.]+)`),
	uaNamed("MIUI Browser", `miuibrowser/([\w.]+)`),
	uaNamed("Mobile Firefox", `fxios/([\w.]+)`),
	uaNamed("WeChat", `micromessenger/([\w.]+)`),
	uaNamed("Baidu", `baiduboxapp/([\w.]+)`),
	uaNamed("UCBrowser", `uc ?browser/([\w.]+)`),
	uaNamed("QQBrowser", `qqbrowser/([\w.]+)`),
	uaNamed("HeyTap", `heytapbrowser/([\w.]+)`),
	uaNamed("Vivo Browser", `vivobrowser/([\w.]+)`),
	uaNamed("Huawei Browser", `huaweibrowser/([\w.]+)`),
	uaNamed("Samsung Internet", `samsungbrowser/([\w.]+)`),
	uaNamed("Konqueror", `konqueror/([\w.]+)`),
	uaNamed("Maxthon", `maxthon/([\w.]+)`),
	uaNamed("Smart Lenovo Browser", `slbrowser/([\w.]+)`),
	uaNamed("Kindle", `kindle/([\w.]+)`),
	// Naver's Whale engine appears inside other Naver apps; only the
	// standalone browser is named.
	{
		re: regexp.MustCompile(`(?i)whale/([\w.]+)`), unless: regexp.MustCompile(`(?i)naver`),
		apply: func(_ string, m []string, res map[string]string) bool {
			res["name"] = "Whale"
			uaSetVersion(res, m[1])
			return true
		},
	},
	uaCapNamed(`(vivaldi|brave|seamonkey|epiphany|falkon|phantomjs|silk|waterfox|librewolf|palemoon)/v?([\w.]+)`),
	uaNamed("NAVER", `naver\(.*?(\d[\d.]*)\)`),
	uaNamed("Instagram", `instagram[ /]([\w.]+)`),
	uaNamed("Facebook", `fbav/([\w.]+);`),
	uaNamed("TikTok", `musical_ly[/_]?([\w.]+)`),
	uaNamed("Electron", `electron/([\w.]+) safari`),
	uaNamed("Chrome Headless", `headlesschrome(?:/([\w.]+))?`),
	uaNamed("Chrome WebView", `\bwv\).+?chrome/([\w.]+)`),
	uaNamed("Mobile Chrome", `\bchrome/([\w.]+).*mobile`),
	uaNamed("Chrome", `\bchrom(?:e|ium)/([\w.]+)`),
	uaNamed("Android Browser", `android.+version/([\w.]+).+?mobile ?safari`),
	uaNamed("Mobile Safari", `version/([\w.]+)(?: .+?)? mobile(?:/\w+)? ?safari`),
	uaNamed("Safari", `version/([\w.]+).+?safari`),
	// A Safari token with no Version/ product is a shell or non-Safari
	// WebKit build; the browser reports as Safari 1.
	{re: regexp.MustCompile(`(?i)safari/[\w.]+`), apply: func(_ string, m []string, res map[string]string) bool {
		res["name"] = "Safari"
		res["version"] = "1"
		return true
	}},
	uaNamed("WebKit", `applewebkit/([\w.]+)`),
	uaNamed("Mobile Firefox", `\bmobile\b.*firefox/([\w.]+)|firefox/([\w.]+).*\bmobile\b`),
	uaCapNamed(`(firefox)/([\w.]+)`),
}

// uaEngineRules identify the rendering engine. Chromium builds before 28
// still shipped WebKit, so the Blink rule requires version 28 or later.
var uaEngineRules = []uaRule{
	uaNamed("EdgeHTML", ` edge/([\w.]+)`),
	uaNamed("Trident", `trident/([\w.]+)`),
	uaNamed("Presto", `presto/([\w.]+)`),
	uaNamed("Goanna", `goanna/([\w.]+)`),
	uaNamed("Blink", `\b(?:headlesschrome|chrom(?:e|ium))/(2[89][\w.]*|[3-9]\d[\w.]*|\d{3,}[\w.]*)`),
	uaNamed("Gecko", `rv:([\w.]{1,9})\b.+gecko`),
	uaNamed("KHTML", `khtml/v?([\w.]+)`),
	uaNamed("WebKit", `applewebkit/([\w.]+)`),
}

// uaWindowsVersions maps Windows NT kernel versions to marketing names.
var uaWindowsVersions = map[string]string{
	"10.0": "10", "6.3": "8.1", "6.2": "8", "6.1": "7", "6.0": "Vista",
	"5.2": "XP", "5.1": "XP", "5.0": "2000", "4.0": "NT", "3.51": "NT",
}

// uaWebOSVersions maps the Chromium major bundled with each LG webOS TV
// release to the webOS version; other majors report the generic TV.
var uaWebOSVersions = map[string]string{
	"38": "3", "53": "4", "68": "5", "79": "6", "87": "22", "94": "23",
	"108": "24", "120": "25",
}

// uaOSRules identify the operating system name and version.
var uaOSRules = []uaRule{
	uaOSNamed("Windows Phone", `windows phone(?: os)? ?([\d.]*)`),
	{re: regexp.MustCompile(`(?i)\bxbox\b(?:.*?\bxbox (one|series [xs]|360))?`), apply: func(_ string, m []string, res map[string]string) bool {
		res["name"] = "Xbox"
		uaSetVersion(res, m[1])
		return true
	}},
	{re: regexp.MustCompile(`(?i)windows nt ([\d.]+)`), apply: func(_ string, m []string, res map[string]string) bool {
		res["name"] = "Windows"
		uaSetVersion(res, uaWindowsVersions[m[1]])
		return true
	}},
	uaOSNamed("Windows", `windows (98|95|me|ce)`),
	// Opera Mini for iOS keeps the J2ME token in its transcoder identity.
	uaOSNamed("iOS", `j2me/midp; opera mini`),
	uaOSNamed("iOS", `\bcpu(?: iphone)? os ([\w_]+) like mac`),
	uaOSNamed("macOS", `(?:ppc |intel )?mac os x ?([\w_.]*)`),
	uaOSNamed("HarmonyOS", `harmonyos`),
	uaOSNamed("Android", `android[ /-]?([\w.]*)`),
	uaOSNamed("Chrome OS", `cros \S+ ([\d.]+)`),
	{re: regexp.MustCompile(`(?i)web0s(?:.*?chrome/(\d+))?`), apply: func(_ string, m []string, res map[string]string) bool {
		res["name"] = "webOS"
		if v, ok := uaWebOSVersions[m[1]]; ok {
			res["version"] = v
		} else {
			res["version"] = "TV"
		}
		return true
	}},
	uaOSNamed("Tizen", `tizen[ /]?([\d.]*)`),
	uaOSNamed("PlayStation", `playstation;? ?(?:playstation )?(\d)`),
	uaCapNamed(`(nintendo) (switch|3ds|wiiu?)`),
	uaOSNamed("BlackBerry", `\bbb(10)\b`),
	uaOSNamed("RIM Tablet OS", `rim tablet os ([\w.]+)`),
	uaOSNamed("BlackBerry", `blackberry\w*/([\w.]+)`),
	uaOSNamed("Symbian", `symbian ?os/([\w.]+)`),
	uaOSNamed("Symbian", `series ?60/([\w.]+)`),
	uaOSNamed("Symbian", `\bs60\b|symbos|symbian`),
	uaCapNamed(`\b(ubuntu|fedora|debian|gentoo|mint|manjaro|opensuse|red ?hat|centos)\b()`),
	{re: regexp.MustCompile(`(?i)\b((?:free|open|net|dragonfly)bsd)[/ ]?(\w*)`), apply: func(_ string, m []string, res map[string]string) bool {
		res["name"] = m[1]
		uaSetVersion(res, uaArchlessVersion(m[2], "amd", "i386", "i686", "x86"))
		return true
	}},
	{re: regexp.MustCompile(`(?i)linux[/ ]?(\w*)`), apply: func(_ string, m []string, res map[string]string) bool {
		res["name"] = "Linux"
		uaSetVersion(res, uaArchlessVersion(m[1], "x86", "arm", "ppc"))
		return true
	}},
}

// uaArchlessVersion filters an OS "version" token that is really a CPU
// architecture: tokens starting with any of the given prefixes are dropped.
func uaArchlessVersion(v string, prefixes ...string) string {
	lower := strings.ToLower(v)
	for _, p := range prefixes {
		if strings.HasPrefix(lower, p) {
			return ""
		}
	}
	return v
}

// uaGenericAndroid pulls the device model out of an Android comment block:
// the last semicolon segment, allowing for a Build/ suffix and a trailing wv
// marker. uaNonModel rejects segments that cannot be a model name: locale
// tags, the wv/Mobile/Tablet markers, and rv: revisions.
var (
	uaGenericAndroid = regexp.MustCompile(`(?i)android[\w. ]*; (?:[^;)]*; )*?([^;)]+?)(?: build/[^;)]*)?(?:; wv)?\)`)
	uaNonModel       = regexp.MustCompile(`(?i)^(?:mobile|tablet|wv|u|rv:.*|[a-z]{2}(?:-[a-z]{2})?|arm)$`)
	uaMobileToken    = regexp.MustCompile(`(?i)\bmobile\b`)
)

// uaDeviceRules identify the device model, vendor and type.
var uaDeviceRules = []uaRule{
	// Samsung: SM-T/X/P are tablets, other SM/SCH/GT/SGH/SHV codes phones.
	// A revision digit after a trailing R is not part of the model name.
	{re: regexp.MustCompile(`(?i)\b(sm-t\w+r)\d*\b`), apply: func(_ string, m []string, res map[string]string) bool {
		res["model"] = m[1]
		res["vendor"] = "Samsung"
		res["type"] = "tablet"
		return true
	}},
	uaDevice(`\b(sm-[txp]\w+)\b`, "", "Samsung", "tablet"),
	uaDevice(`\b(sm-\w+|sch-\w+|gt-\w+|sgh-\w+|shv-\w+)\b`, "", "Samsung", "mobile"),
	{re: regexp.MustCompile(`(?i)nintendo (switch|3ds|wiiu?)`), apply: func(_ string, m []string, res map[string]string) bool {
		res["model"] = m[1]
		res["vendor"] = "Nintendo"
		res["type"] = "console"
		return true
	}},
	uaDevice(`microsoft; (lumia[\w ]+?)[;)]`, "", "Microsoft", "mobile"),
	uaDevice(`(lumia[\w ]+?)[;)]`, "", "Nokia", "mobile"),
	uaDevice(`\b(ipad)\b`, "iPad", "Apple", "tablet"),
	uaDevice(`\b(ipod touch)\b`, "iPod touch", "Apple", "mobile"),
	uaDevice(`\b(iphone|ipod)\b`, "iPhone", "Apple", "mobile"),
	uaDevice(`(macintosh)`, "Macintosh", "Apple", ""),
	// Huawei model codes appear with or without the maker's name.
	uaDevice(`\b(?:huawei[ _-]?)?((?:vog|eml|noh|jny|ane|ele|lya|mar|pot|stk|yal|clt|col|pra|eva|vtr|was|fig|bla|par|ine)-[a-z]*\d+\w*)\b`, "", "Huawei", "mobile"),
	uaDevice(`huawei[ _-]?([\w-]+?(?: [\w]+?)*?)(?: build|[;)])`, "", "Huawei", "mobile"),
	// Xiaomi: MI/Redmi/POCO names, M-prefixed codes, and the bare numeric
	// codes used since 2021 (year, month, sequence, region letter).
	uaDevice(`\b(redmi[\w ]+?|mi \w+|poco[\w ]*?|m2\d{3}[a-z]\w*|2\d{6,7}[cgi])(?: build|[;)])`, "", "Xiaomi", "mobile"),
	// OnePlus phones use their own code families plus, since 2023, OPPO's
	// CPH prefix with an odd number from 2301 up.
	uaDevice(`\b(?:oneplus ?)?(a\d{4}|gm\d{4}|hd\d{4}|le2\d{3}|ac2\d{3}|in2\d{3}|kb2\d{3}|ne2\d{3}|mt2\d{3}|be2\d{3}|cph2[3-6]\d[13579]|p[ghj][a-z]{1,2}\d{3})\b`, "", "OnePlus", "mobile"),
	uaDevice(`\b(opd2\d{3})\b`, "", "OPPO", "tablet"),
	uaDevice(`\b(cph\d{4}|pe[a-z]m\d{2})\b`, "", "OPPO", "mobile"),
	uaDevice(`\b(rmx\d{4})\b`, "", "Realme", "mobile"),
	// Google Pixels; only the tablet is not a phone.
	{re: regexp.MustCompile(`(?i)\b(pixel[\w ]*?)(?: build|[;)])`), apply: func(_ string, m []string, res map[string]string) bool {
		res["model"] = m[1]
		res["vendor"] = "Google"
		if strings.Contains(strings.ToLower(m[1]), "tablet") {
			res["type"] = "tablet"
		} else {
			res["type"] = "mobile"
		}
		return true
	}},
	uaDevice(`motorola[ -]([\w ]+?)(?: build|[;)])`, "", "Motorola", "mobile"),
	uaDevice(`\b(lm-\w+)\b`, "", "LG", "mobile"),
	uaDevice(`\blenovo ?(tb[-\w]+|yt[-\w]+)\b`, "", "Lenovo", "tablet"),
	uaDevice(`nokia[ _-]?([\w-]+?)(?:[/;)]| build)`, "", "Nokia", "mobile"),
	uaDevice(`\bitel (\w+)`, "", "itel", "mobile"),
	// T-Mobile's REVVL line; the TAB models are tablets.
	{re: regexp.MustCompile(`(?i)\b(revvl[\w ]*?)(?: build|[;)])`), apply: func(_ string, m []string, res map[string]string) bool {
		res["model"] = m[1]
		res["vendor"] = "T-Mobile"
		if strings.Contains(strings.ToLower(m[1]), "tab") {
			res["type"] = "tablet"
		} else {
			res["type"] = "mobile"
		}
		return true
	}},
	// Amazon: Fire TV codes carry an AFT prefix that is not part of the
	// model; Fire tablets use KF codes; the Kindle browser reports itself.
	uaDevice(`\baft(\w+)\b`, "", "Amazon", "smarttv"),
	uaDevice(`\b(kf[a-z0-9]+)\b`, "", "Amazon", "tablet"),
	{re: regexp.MustCompile(`(?i)(kindle)/([\w.]+)`), apply: func(_ string, m []string, res map[string]string) bool {
		res["model"] = m[2]
		res["vendor"] = "Kindle"
		res["type"] = "tablet"
		return true
	}},
	uaDevice(`\b(playstation [345](?: pro)?)\b`, "", "Sony", "console"),
	uaDevice(`\b(xbox (?:one|series [xs]|360))\b`, "", "Microsoft", "console"),
	uaDevice(`roku\w*/(dvp-[\w.]+)`, "", "Roku", "smarttv"),
	uaDevice(`apple ?tv`, "Apple TV", "Apple", "smarttv"),
	uaDevice(`\bbb10; (\w+)`, "", "BlackBerry", "mobile"),
	uaDevice(`(playbook)`, "PlayBook", "RIM", "tablet"),
	uaDevice(`blackberry ?(\w+)/`, "", "BlackBerry", "mobile"),
	uaDevice(`\bsmart[- ]?tv\b|web0s`, "", "", "smarttv"),
	// Any other Android device: the model is the last comment segment; a
	// Mobile token outside the comment separates phones from tablets.
	{re: uaGenericAndroid, apply: func(ua string, m []string, res map[string]string) bool {
		model := strings.TrimSpace(m[1])
		if uaNonModel.MatchString(model) {
			return false
		}
		res["model"] = model
		if uaMobileToken.MatchString(ua) {
			res["type"] = "mobile"
		} else {
			res["type"] = "tablet"
		}
		return true
	}},
	// With no model at all, a Mobile token still gives the type.
	uaDevice(`\bmobile\b`, "", "", "mobile"),
}

// uaCPURules identify the CPU architecture. Where the family name itself is
// the output, the matched token is reported lowercased.
var uaCPURules = []uaRule{
	{re: regexp.MustCompile(`(?i)\b(?:amd64|x86_64|x64|wow64|win64)\b`), apply: func(_ string, m []string, res map[string]string) bool {
		res["architecture"] = "amd64"
		return true
	}},
	{re: regexp.MustCompile(`(?i)\b(?:i[36]86|x86|ia32)\b`), apply: func(_ string, m []string, res map[string]string) bool {
		res["architecture"] = "ia32"
		return true
	}},
	{re: regexp.MustCompile(`(?i)\b(?:aarch64|arm64|armv8\w*)\b`), apply: func(_ string, m []string, res map[string]string) bool {
		res["architecture"] = "arm64"
		return true
	}},
	{re: regexp.MustCompile(`(?i)\barm(?:v[67]\w*)?\b`), apply: func(_ string, m []string, res map[string]string) bool {
		res["architecture"] = "arm"
		return true
	}},
	{re: regexp.MustCompile(`(?i)\b(ppc64|ppc|powerpc)\b`), apply: func(_ string, m []string, res map[string]string) bool {
		res["architecture"] = strings.ToLower(strings.Replace(m[1], "powerpc", "ppc", 1))
		return true
	}},
	{re: regexp.MustCompile(`(?i)\b(mips64|mips)\b`), apply: func(_ string, m []string, res map[string]string) bool {
		res["architecture"] = strings.ToLower(m[1])
		return true
	}},
	{re: regexp.MustCompile(`(?i)\b(sparc64|sparc)\b`), apply: func(_ string, m []string, res map[string]string) bool {
		res["architecture"] = strings.ToLower(m[1])
		return true
	}},
}
