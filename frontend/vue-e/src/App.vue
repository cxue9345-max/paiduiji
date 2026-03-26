<template>
    <div id="app">
      <div id="nav" hidden>
        <router-link to="/test">Test</router-link>
        <router-link to="/test1">Test1</router-link>
      </div>
      <router-view/>
    </div>
</template>
<script>

export default {
  name: 'App',
  components: {
    // HelloWorld
  }
}
</script>

<style>
</style>
