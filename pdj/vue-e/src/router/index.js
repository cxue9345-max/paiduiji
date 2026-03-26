import Vue from 'vue'
import Router from 'vue-router'
import Test from '@/views/Test'
import Test1 from '@/views/Test1'

Vue.use(Router)

export default new Router({
    mode: "history",
    routes: [
        {
            path: '/test',
            name: 'test',
            component: Test
        },
        {
            path: '/test1',
            name: 'test1',
            component: Test1
        }
    ]
  })