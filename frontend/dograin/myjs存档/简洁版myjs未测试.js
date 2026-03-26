var admins =["迷糊的迷糊菇","一纸轻予梦","你需要写谁的名字就写谁"];
//上面是b站管理员的用户名，必须填写
//以下无需更改，改不改没有意义。
var zroomid =26714219;
var zuid = 18461303;
var zbuvid = "";
var persons =[];
var ws = new WebSocket ("ws://localhost:23333/danmu/sub");

ws.onopen = function () {
    document.getElementById("status").append("已连接");
    ws.send(encode(JSON.stringify({
        uid: zuid,
        roomid: zroomid,
        buvid:zbuvid,
                 
    }), 7));
}
var shujubaobao;

console.log("[打算借壳重生]");
ws.onmessage = async function (msgEvent) {
    console.log(msgEvent.data)
    shujubaobao = msgEvent.data    
    jsontoprint(JSON.parse(shujubaobao));
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

            if(index != -1){
                if(message=="取消排队"){
                    persons.splice(index,1);
                }            
                var TiHuanIdex = message.indexOf("替换");

                if(TiHuanIdex >= 0&&TiHuanIdex<=4||XiuGaiIdex>= 0&&XiuGaiIdex<=4){
                    if(TiHuanIdex >= 0){
                        var Message_change=message.replace("替换","");}              
                    var TiHuanMessage ="<span>"+person + Message_change+"<br></span>"
                        persons.splice(index,1,TiHuanMessage);
                    } 
                }

            var amminIdex =admins.indexOf(person);
            if(amminIdex!=-1){

                var delIdex = message.indexOf("del");
                if(delIdex >= 0&&delIdex<=4){
                    var dindex=message.replace(/[^0-9]/ig,"");//除了数字 全部删除
                    //条件 长度大于0，删除数字以外后剩下非空值，序号的长度大于等于需要删除的位置。
                    if(dindex!=""&&persons.length>=dindex&&persons.length>0){
                       persons.splice(dindex-1,1);
                    };
                };

                var newIdex = message.indexOf("add");

                if(newIdex >= 0&&newIdex <= 4||xinzengIdex >= 0 && xinzengIdex <=4||tianjiaIdex >= 0 && tianjiaIdex <=4){
                    var add_message = message.replace("add","");
                    if(add_message!=""){
                        persons.push("<span>"+add_message+"<br></span>");
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



