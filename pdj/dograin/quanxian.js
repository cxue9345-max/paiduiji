//修改 此处 无效

//设置内容
//------------------------------------------------------------------------------------------
var admins =["迷糊的迷糊菇","一纸轻予梦","写一下你的名字哦"];//管理员
var ban_admins = ["黑名单成员","黑名单成员2"]; //禁止排队但不禁言
var jianzhang = [];//舰长插队
var fankui = false; //是否开启反馈，就是排上了要不要发弹幕(不建议开)
var guanli_fankui = false; //是否开启反馈，就是管理员操作了要不要发弹幕(不建议开)
var paidui_list_length_max = 100 ; //排队人数上限，默认100，直播中可通过指令修改。
var jianzhangchadui = false; //是否开启舰长插队
var jianzhang_cd_kind = 1; //舰长插队类型|1:默认(已实装);2:上舰插队(未实装);3:礼物插队(未实装);|
var jianzhang_cd_cishu = 1;//被舰长插队的普通人次数，比如最多被插队一次；//未实装
var fangguan_can_doing = false //房管默认为插件控制者(主播已默认)
var all_suoyourenbukepaidui = false //所有人禁止排队,当然仅主播管理员可使用
var YHbot_kaiguan = false; //如需使用改成true
var YHbotId = "";//云湖app
var YHbot_msg_type = ""; //云湖app
var YHbot_webhook_token = "";//云湖app
var QYWX_kaiguan = false;//企业微信开关
var WX_webhook = ""; //企业微信
var only_myfuns_paidui = false //仅本直播间牌子可排队(需要设置,未实装)。
var liwu_chadui_kg = false;//礼物插队
var liwu_paidui_kg = false;//礼物排队
var liwu_chadui_kind = 50;//插队礼物
var liwu_chadui_kind = 50;//排队礼物


//-----------------------------------------------------------------------------
jianzhang.push("一纸轻予梦");//强制授予 一纸轻予梦的舰长身份，用于测试。
console.log("[基础信息]","管理员:",admins,"黑名单",ban_admins,"初始舰长权限",jianzhang);
//预写入-
super_xinxi_desu = [["一纸轻予梦","喜欢雨露的小青草儿"],[0,0],[0,0],[],[],[],[],[],[],[]]
// 0 名字
// 1 当天排队数
// 2 当天被插队数
// 3 
// 4 
// 5