<template>
  <el-container id="cont" style=" border: 1px solid #eee">
  <el-aside width="20%" style="background-color: rgb(238, 241, 246)">
      <div class="vText">
            <div>
                <span>弹幕【排队】【取消排队】</span>
            </div>
      </div>
  </el-aside>
  
  <el-container>

    <div   id='status' class="div" hidden>
    </div>
    <div   id='popularity' class="div" hidden>
    </div>
    <div   id='danmu' class="div">
    </div>
  
  </el-container>
</el-container>
</template>

<script>


 export default {
   name: "test",
   mounted() {
     console.log(this.$demo.persons);
  },
}

</script>

<style lang="less" scoped>
  .div {
      height: 90%;
      line-height:1.7;
      font-size: 90px;
      float: left;
      margin-top: 5%;
      text-align:left;
      overflow: hidden; 
  }

  .div span {
      background: linear-gradient(to right, red, blue);
      -webkit-background-clip: text;
      color: transparent;
  }

  .vText {
      float: left;
      height: 100%;
      width: 100%;
      /* text-overflow:ellipsis;超出部分显示省略号 */
      white-space: nowrap;/*规定段落中的文本不进行换行 */
      text-align:center;
      font-size: 30px;
     
  }

  .vText div{
      height: 70%;
      display: initial;
      overflow: hidden;
      writing-mode:tb-rl; 
  }
  .vText span {
      margin-top: 5%;
      background: linear-gradient(to right, blue, red);
      -webkit-background-clip: text;
      color: transparent;
  }

  #cont {
    width: 400px;
    height: 500px;
  }
</style>
