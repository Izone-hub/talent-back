'use client'

export default function Greeting({ name = 'World' }) {
  return <p data-test="greeting">Hello, {name}!</p>
}
