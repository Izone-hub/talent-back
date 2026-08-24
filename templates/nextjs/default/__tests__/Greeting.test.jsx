import { describe, expect, it } from 'vitest'
import { render, screen } from '@testing-library/react'
import Greeting from '../components/Greeting'

describe('Greeting', () => {
  it('greets the given name', () => {
    render(<Greeting name="Ada" />)
    expect(screen.getByText('Hello, Ada!')).toBeTruthy()
  })

  it('falls back to World', () => {
    render(<Greeting />)
    expect(screen.getByText('Hello, World!')).toBeTruthy()
  })
})
