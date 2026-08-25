import { describe, expect, it } from 'vitest'
import { mount } from '@vue/test-utils'
import HelloWorld from '../src/HelloWorld.vue'

describe('HelloWorld', () => {
  it('increments the counter', async () => {
    const wrapper = mount(HelloWorld)
    expect(wrapper.find('[data-test="count"]').text()).toContain('Count: 0')
    await wrapper.find('[data-test="inc"]').trigger('click')
    expect(wrapper.find('[data-test="count"]').text()).toContain('Count: 1')
  })
})
