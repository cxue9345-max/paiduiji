var admins =["迷糊的迷糊菇","一纸轻予梦","你需要写谁的名字就写谁"];
//上面是b站用户名，必须填写
//怎么说呢，借壳重生的版本，只需要写管理员就行了。
//以下无需更改，改不改没有意义了。
//版本号：20240121

var zroomid =26714219;
var zuid = 18461303;
var zbuvid = "";
//从上往下：你的直播间号，你的uid，房间管理员名字。
//迷糊uid 629543291
//迷糊房间号 26714219
//一纸轻予梦的18461303
//BUVID 需要自己访问下面的api，然后点击cookie，查看标头。
//https://api.live.bilibili.com/room/v1/Danmu/getConf?room_id=26714219&platform=pc&player=web

//-----------------------------------------------------------------------------

console.log("[宣告初始变量]");
console.log("房间id");
console.log(zroomid);
console.log("主播uid");
console.log(zuid);
console.log("管理员");
console.log(admins);
console.log("Buvid号");
console.log(zbuvid);
if (zbuvid == null){
    console.log("buvid是空值进去?你先试试能不能联吧，不行再回去抓包，填写。当然，这个不写不一定不行。");
}

var persons =[];
//新建链接时会自动连接，并自动调用onopen方法
//var ws = new WebSocket("wss://broadcastlv.chat.bilibili.com/sub");原网址，直连b站
//var ws = new WebSocket("wss://broadcastlv.chat.bilibili.com/sub");
//var ws = new WebSocket ("ws://localhost:23333/danmu/sub");借用别人的wss
var ws = new WebSocket ("ws://localhost:23333/danmu/sub");

var URL_token = "https://api.live.bilibili.com/room/v1/Danmu/getConf?room_id=26714219&platform=pc&player=web";
if (zroomid !== 26714219){
    //宣告为数字类型
    //var afangjian_id = Number;
    //赋值为房间号
    let afangjian_id = zroomid;
    //转化为整形
    afangjian_id  = afangjian_id.toString();
    var first_url_token = "https://api.live.bilibili.com/room/v1/Danmu/getConf?room_id=";
    var end_url_token = "&platform=pc&player=web";
    URL_token = first_url_token + afangjian_id + end_url_token;
    console.log("获取的URL，不知道有没有错误")
    console.log(URL_token)

}
//借壳重生了，以下内容不重要，填不填无所谓了，
//20240120留！
ws.onopen = function () {
    document.getElementById("status").append("已连接");
    ws.send(encode(JSON.stringify({
        uid: zuid,
        //uid:zuid,
        roomid: zroomid, /* 22646908*/
        //protover:3,
        //platform: "web",   
        //type:2,
        buvid:zbuvid,
                 
    }), 7));
}

//{"uid": 0表示未登录，否则为用户ID,"roomid": 房间ID,"protover": 1,"platform": "web","clientver": "1.4.0"}
var shujubaobao;

console.log("[打算借壳重生]");
ws.onmessage = async function (msgEvent) {

    //var read = new FileReader();
    //read.readAsArrayBuffer(msgEvent.data)//从消息里面拿出数据包，旧版本
    console.log(msgEvent.data)
    shujubaobao = msgEvent.data
    
    jsontoprint(JSON.parse(shujubaobao));//调用下面的函数

    //以下部分代码再无作用，但是感谢之前大佬写的，不删除！
    //read.onload = function (eve) {
    //    let buffer = new Uint8Array(eve.target.result)//将二进制包转换成8位无符号整型数组
    //    let ver = readInt(buffer, 6, 2)
    //    let op = readInt(buffer, 8, 4)
    //    switch (ver) {
    //        case 0:
    //            console.log("[case = 0]----------");
    //            let body = textDecoder.decode(buffer.slice(16))
    //            if (body)
    //                console.log(JSON.parse(body));
    //            break;
    //        case 1:
    //            console.log("[case = 1]----------");
    //            if (op == 8) {
    //                console.log('认证成功开始心跳')
    //                ws.send(encode('[object Object]', 2));
    //                setInterval(function () {
    //                    ws.send(encode('[object Object]', 2));
    //                }, 30000);
    //            }
    //            else {
    //                let packetLen = readInt(buffer, 0, 4) - 16;
    //                let pop = readInt(buffer, 16, packetLen)
    //                console.log('气人值：' + pop)
    //                document.getElementById("popularity").innerText='气人值：'+pop
    //            }
    //            break;
    //        case 2:
    //            console.log("[case = 2]----------");
    //            let data = 0
    //            try {
    //                data = pako.inflate(buffer.slice(16))
    //                let last = data.byteLength
    //                while (last > 0) {
    //                    let packetLen = readInt(data, 0, 4)
    //                    let body = textDecoder.decode(data.slice(16, packetLen))
    //                    if (body)
    //                        jsontoprint(JSON.parse(body));
    //                    data = data.slice(packetLen)
    //                    last = last - packetLen
    //                }
    //            } catch (error) {
//
    //            }
//
//
//                break;
//            default:
//                console.log('error')
////                break;
//        }
//
//
//    }
//
};


const textEncoder = new TextEncoder('utf-8');
const textDecoder = new TextDecoder('utf-8');

const readInt = function (buffer, start, len) {
    let result = 0
    for (let i = len - 1; i >= 0; i--) {
        result += Math.pow(256, len - i - 1) * buffer[start + i]
    }
    return result
}

const writeInt = function (buffer, start, len, value) {
    let i = 0
    while (i < len) {
        buffer[start + i] = value / Math.pow(256, len - i - 1)
        i++
    }
}

const encode = function (str, op) {
    let data = textEncoder.encode(str);
    let packetLen = 16 + data.byteLength;
    let header = [0, 0, 0, 0, 0, 16, 0, 1, 0, 0, 0, op, 0, 0, 0, 1]
    writeInt(header, 0, 4, packetLen)
    return (new Uint8Array(header.concat(...data))).buffer
}

//
function jsontoprint(data) {
    var indexBool_danmu = data.cmd.indexOf('danmu'); 
    var indexBool_DANMU_MSG = data.cmd.indexOf('DANMU_MSG');
    var indexBool_SUPER_CHAT_MESSAGE = data.cmd.indexOf('SUPER_CHAT_MESSAGE');
    

    var indexContent ;
    if(indexBool_danmu >= 0 ){
        indexContent = 'danmu';
    }

    if(indexBool_DANMU_MSG >= 0 ){
        indexContent = 'DANMU_MSG';
    }

    if(indexBool_SUPER_CHAT_MESSAGE >= 0 ){
        indexContent = 'SUPER_CHAT_MESSAGE';
    }

    switch (indexContent) {
            case 'danmu': //case 'DANMU_MSG':

            var message =data.result.msg;            ;
            var person = data.result.uname;

            //message = 原始信息
            //person = 权限组

            
            var index =persons.findIndex((item,index) => { return item.indexOf(person) != -1 });

            console.log("person:"+person);
            console.log("persons:"+persons);
            console.log("index:"+index);
            console.log("admins:"+admins);
            
            
            if(message=="排队"){                
                if(index < 0){
                    persons.push("<span>"+person+"<br></span>");
                    console.log(person);
                }               
            }
            
            if(message=="官服排"){                
                if(index < 0){
                    persons.push("<span>[官]"+person+"<br></span>");
                    console.log(person);
                }               
            }

            if(message=="B服排"){                
                if(index < 0){
                    persons.push("<span>[B]"+person+"<br></span>");
                    console.log(person);
                }               
            }

            if(message=="超级排"){                
                if(index < 0){
                    persons.push("<span><"+person+"><br></span>");
                    console.log(person);
                }               
            }

            var messageFlag =message.indexOf("排队 ");
            console.log("message:"+message);
            
            if(messageFlag != -1){
                var messageIndex = message.indexOf(" ");
                console.log("messageIndex:"+messageIndex);
                var massageSub =  message.slice(messageIndex);
                console.log("massageSub:"+massageSub);
                if(index < 0){
                    persons.push("<span>"+person+massageSub+"<br></span>");
                }
            }

            var messageFlag_1 =message.indexOf("超级排 ");
            console.log("message:"+message);
            
            if(messageFlag_1 != -1){
                var messageIndex_1 = message.indexOf(" ");
                var massageSub_1 =  message.slice(messageIndex_1);

                massageSub_1 = ":" + massageSub_1 + ""
                Superperson = "<" + person + ">"

                console.log("messageIndex:"+messageIndex_1);
                console.log("massageSub:"+massageSub_1);
                if(index < 0){
                    persons.push("<span>"+Superperson+massageSub_1+"<br></span>");
                }
            }
            
            var messageFlag_guanfu =message.indexOf("官服排 ");
            console.log("message:"+message);
            
            if(messageFlag_guanfu != -1){
                var messageIndex_guanfu = message.indexOf(" ");
                var massageSub_guanfu =  message.slice(messageIndex_guanfu);

                massageSub_guanfu = "" + massageSub_guanfu + ""
                SuperpersonG = "[官]" + person + ""

                console.log("messageIndex:"+messageIndex_guanfu);
                console.log("massageSub:"+massageSub_guanfu);
                if(index < 0){
                    persons.push("<span>"+SuperpersonG+massageSub_guanfu+"<br></span>");
                }
            }

            var messageFlag_Bfu =message.indexOf("B服排 ");
            console.log("message:"+message);
            
            if(messageFlag_Bfu != -1){
                var messageIndex_Bfu = message.indexOf(" ");
                var massageSub_Bfu =  message.slice(messageIndex_Bfu);

                massageSub_Bfu = "" + massageSub_Bfu + ""
                SuperpersonB = "[B]" + person + ""

                console.log("messageIndex:"+messageIndex_Bfu);
                console.log("massageSub:"+massageSub_Bfu);
                if(index < 0){
                    persons.push("<span>"+SuperpersonB+massageSub_Bfu+"<br></span>");
                }
            }
            //已经排上队的人（列表有名字）
            if(index != -1){
                if(message=="取消排队"||message=="排队取消"||message=="我确认我取消排队"){
                    persons.splice(index,1);
                }
            
                var TiHuanIdex = message.indexOf("替换");
                var XiuGaiIdex = message.indexOf("修改");
                if(TiHuanIdex >= 0&&TiHuanIdex<=4||XiuGaiIdex>= 0&&XiuGaiIdex<=4){
                    if(TiHuanIdex >= 0){
                        var Message_change=message.replace("替换","");}
                    if(XiuGaiIdex >= 0){
                        var Message_change=message.replace("修改","");}               
                    var TiHuanMessage ="<span>"+person + Message_change+"<br></span>"
                        persons.splice(index,1,TiHuanMessage);
                    } 
                }
            //已排之人权限末尾
            
            //管理员权限
            var amminIdex =admins.indexOf(person);
            if(amminIdex!=-1){
                //.indexOf是查找字符的位置
                //找到了，返回字符所在位置

                //删列表系列
                var delIdex = message.indexOf("del");
                var ShanCIdex = message.indexOf("删除");
                var WanCIndex = message.indexOf("完成");
                if(delIdex >= 0&&delIdex<=4 || ShanCIdex >= 0 && ShanCIdex <=4||WanCIndex >= 0&&WanCIndex <=4){
                    var dindex=message.replace(/[^0-9]/ig,"");//除了数字 全部删除
                    //条件 长度大于0，删除数字以外后剩下非空值，序号的长度大于等于需要删除的位置。
                    if(dindex!=""&&persons.length>=dindex&&persons.length>0){
                       persons.splice(dindex-1,1);
                    };
                    //条件 长度大于等于1，删除数字以外后剩下是空值。
                    //if(dindex==""&&persons.length>0){
                    //    persons.splice(0,1);
                    //} 
                };
                
                //新增模块：add/新增/添加，顺序添加
                var newIdex = message.indexOf("add");
                var xinzengIdex = message.indexOf("新增");
                var tianjiaIdex = message.indexOf("添加");
                if(newIdex >= 0&&newIdex <= 4||xinzengIdex >= 0 && xinzengIdex <=4||tianjiaIdex >= 0 && tianjiaIdex <=4){
                    var add_message = message.replace("add","");
                    add_message=add_message.replace("add","");
                    add_message=add_message.replace("新增","");
                    add_message=add_message.replace("添加","");
                    if(add_message!=""){
                        persons.push("<span>"+add_message+"<br></span>");
                    } 
                };
                //无影插队系列
                var WYCDIndex = message.indexOf("无影插");
                if (WYCDIndex >= 0 && WYCDIndex <= 4) {
                  // 使用正则表达式匹配数字部分，例如 "无影插 2 胡桃" 中的 "2"
                  var WYpositionMatch = message.match(/\d+/);
                  
                  // 如果找到了匹配的位置
                  if (WYpositionMatch) {
                    var position = parseInt(WYpositionMatch[0], 10); // 将匹配的位置字符串转换为整数
                    
                    if (position >= 1 && position <= 20) {
                      // 删除 "无影" 和数字部分，只保留内容
                      var WYchaduiText = message.replace(/无影插\s+\d+\s+/, '');
                      WYchaduiText = "<span>" + WYchaduiText + "<br></span>"
                      // 在指定位置插入新内容
                      persons.splice(position - 1, 0, WYchaduiText);
                    }
                  }
                };
                //插队系列
                var chaduiIndex = message.indexOf("插队");
                if (chaduiIndex >= 0 && chaduiIndex <= 4) {
                  // 使用正则表达式匹配数字部分，例如 "插队 2 胡桃" 中的 "2"
                  var positionMatch = message.match(/\d+/);
                  
                  // 如果找到了匹配的位置
                  if (positionMatch) {
                    var position = parseInt(positionMatch[0], 10); // 将匹配的位置字符串转换为整数
                    
                    if (position >= 1 && position <= 30) {
                      // 删除 "插队" 和数字部分，只保留内容
                      var chaduiText = message.replace(/插队\s+\d+\s+/, '');
                      chaduiText = "<span>@" + chaduiText + "<br></span>"
                      // 在指定位置插入新内容
                      persons.splice(position - 1, 0, chaduiText);
                    }
                  }
                }

                //var chaduiIdex = message.indexOf("插队");
                //if(nIdex >= 0){
                    //var dmessage=message.replace("add","");
                    //if(dmessage!=""){
                    //    persons.push("<span>"+dmessage+"<br></span>");
                    //} 
                //}

            }
            
            console.log(persons);
            //let para = document.createElement("p")
            //let node = document.createTextNode(data.info[2][1]+':'+data.info[1])
            //para.appendChild(node)
            document.getElementById('danmu').innerHTML=persons.join("");
            break; 
        case 'SUPER_CHAT_MESSAGE':
            // console.log(data)
            // console.log(data.data.message)
            document.getElementById('danmu').append(data.data.price+'$'+data.data.user_info.uname+':'+data.data.message+'\n')
            break
        case 'DANMU_MSG':
            break
        default:
            // console.log(data)
            //document.getElementById('danmu').append(data.cmd+'\n')
            break
    }
}



