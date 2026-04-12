import { useParams } from "react-router-dom"

export default function StratBoardPage() {
  const { id } = useParams<{ id: string }>()

  return (
    <div>
      <h1 className="text-2xl font-bold">Strategy Board</h1>
      <p className="mt-2 text-muted-foreground">
        Editing board <code className="text-foreground">{id}</code>
      </p>
    </div>
  )
}
