import { useParams } from "react-router-dom";

export default function MemoryDetail() {
  const { slug } = useParams<{ slug: string }>();

  return (
    <div className="p-8">
      <h1 className="text-3xl font-bold">Memory Detail</h1>
      <p className="mt-2 text-muted-foreground">
        Viewing memory: <code>{slug}</code>
      </p>
    </div>
  );
}
