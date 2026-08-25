import { useState } from 'react'

export default function App() {
  const [count, setCount] = useState(0)
  return (
    <div>
      <h1>Hello, Sandbox!</h1>
      <button data-test="inc" onClick={() => setCount(count + 1)}>
        Count: {count}
      </button>
    </div>
  )
}
