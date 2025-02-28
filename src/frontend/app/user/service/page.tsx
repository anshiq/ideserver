"use client";

import Dialog from "@/app/components/Dialog";
import { axiosFetchAuth } from "@/lib/axiosConfig";
import { useRouter } from "next/navigation";
import { useEffect, useState } from "react";

type DataType = {
  _id: string;
  name: string;
  stack: string;
  yamlCode: string;
  iconUrl: string;
};

export default function Service() {
  const router = useRouter()
  const [containers, setContainers] = useState<DataType[]>([]);
  const [dialogOpen, setDialogOpen] = useState(false);
  const [selectedContainer, setSelectedContainer] = useState({
    id: "",
    name: "",
    icon: "",
  });
  const [serviceName, setServiceName] = useState("");

  useEffect(() => {
    const fetchData = async () => {
      try {
        const response = await axiosFetchAuth().get("/get-all-containers");
        setContainers(response.data);
      } catch (error) {
        console.error("Error fetching containers:", error);
      }
    };
    fetchData();
  }, []);

  const handleSubmit = async (e) => {
    e.preventDefault();
    if (selectedContainer.id.length < 10)
      return alert("Error Selecting container");
    try {
     const k =  await axiosFetchAuth().post("/service", {
        containerId: selectedContainer.id,
        name: serviceName,
      });
      setDialogOpen(false);
      if(k.status ==200){
        router.push("/service/:id")
      }
    } catch (error) {
      console.error("Error creating service:", error);
    }
  };

  return (
    <div className="p-6 bg-gray-100 min-h-screen flex items-center justify-center">
      <div className="bg-white p-6 rounded-2xl shadow-xl w-full max-w-3xl">
        <h1 className="text-3xl font-bold text-gray-800 mb-6 text-center">
          Available Containers
        </h1>
        <div className="grid md:grid-cols-2 gap-6">
          {containers.map((item) => (
            <div
              key={item._id}
              className="bg-gray-50 p-4 rounded-xl shadow-md flex flex-col items-center justify-center border hover:shadow-lg transition"
            >
              <div
                dangerouslySetInnerHTML={{ __html: item.iconUrl }}
                className="w-16 h-16 mb-3"
              ></div>
              <h2 className="text-xl font-semibold text-gray-800">
                {item.name}
              </h2>
              <p className="text-gray-600 text-sm">Stack: {item.stack}</p>
              <button
                onClick={() => {
                  setDialogOpen(true);
                  setSelectedContainer({
                    id: item._id,
                    name: item.name,
                    icon: item.iconUrl,
                  });
                }}
                className="mt-4 px-4 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700 transition"
              >
                Run
              </button>
            </div>
          ))}
        </div>
      </div>

      {dialogOpen && (
        <Dialog
          isOpen={dialogOpen}
          onClose={() => setDialogOpen(false)}
          title="Service Details"
        >
          <div className="p-4">
            <div className="flex flex-col items-center text-center">
              <div
                dangerouslySetInnerHTML={{ __html: selectedContainer.icon }}
                className="w-16 h-16 mb-3"
              ></div>
              <h2 className="text-2xl font-semibold text-gray-800">
                {selectedContainer.name}
              </h2>
            </div>
            <form onSubmit={handleSubmit} className="mt-4">
              <label className="block text-gray-700 font-medium mb-1">
                Enter Service Name
              </label>
              <input
                onChange={(e) => setServiceName(e.target.value)}
                required
                type="text"
                className="w-full p-2 border rounded-lg focus:outline-none focus:ring-2 focus:ring-blue-500"
              />
              <button
                type="submit"
                className="mt-4 w-full px-4 py-2 bg-green-600 text-white rounded-lg hover:bg-green-700 transition"
              >
                Create
              </button>
            </form>
          </div>
        </Dialog>
      )}
    </div>
  );
}
