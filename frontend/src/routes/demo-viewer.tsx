import { useParams } from "react-router-dom"

export default function DemoViewerPage() {
  const { id } = useParams<{ id: string }>()

  return (
    <div>
      <h1 className="text-2xl font-bold">Demo Viewer</h1>
      <p className="mt-2 text-muted-foreground">
        Viewing demo <code className="text-foreground">{id}</code>
      </p>
    </div>
  )
}
