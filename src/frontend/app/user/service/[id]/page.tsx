"use client";

import { axiosFetchAuth } from "@/lib/axiosConfig";
import { useEffect, useState } from "react";
import { useParams, useRouter } from "next/navigation";
import Image from "next/image";
import { useUserAuth } from "@/app/components/UserWrapper";

interface Service {
  _id: string;
  name: string;
  status: "terminated" | "active" | "spawning" | "";
  linkedContainer: string;
}

interface Container {
  _id: string;
  name: string;
  stack: string;
  iconUrl: string;
}

export default function GetServiceDetails() {
  const params = useParams();
  const route = useRouter();
  const serviceId = (params?.id as string) || "";
  const { wsConnection } = useUserAuth();

  const [service, setService] = useState<Service | null>(null);
  const [container, setContainer] = useState<Container | null>(null);
  const [loading, setLoading] = useState<boolean>(true);
  const [error, setError] = useState<string | null>(null);
  const [serviceStatus, setServiceStatus] = useState<
    "terminated" | "active" | "spawning" | ""
  >("");
  const [isProcessing, setIsProcessing] = useState<boolean>(false);

  const fetchServiceDetails = async () => {
    setLoading(true);
    setError(null);

    try {
      if (!serviceId) {
        throw new Error("Service ID is required");
      }

      const response = await axiosFetchAuth().get(`/service/${serviceId}`);
      setService(response.data.service);
      setContainer(response.data.container);
      setServiceStatus(response.data.service.status);
    } catch (err) {
      console.error("Error fetching service details:", err);
      setError(
        err instanceof Error ? err.message : "Failed to fetch service details"
      );
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    if (serviceId) {
      fetchServiceDetails();
    }
  }, [serviceId]);

  useEffect(() => {
    let intervalId: NodeJS.Timeout;

    if (serviceStatus === "spawning") {
      wsConnection?.send(
        JSON.stringify({ serviceId: serviceId, type: "serviceStatus" })
      );
    } else if (serviceStatus === "active") {
      intervalId = setInterval(() => {
        wsConnection?.send(
          JSON.stringify({ serviceId: serviceId, type: "serviceStatus" })
        );
      }, 20000);
    }

    return () => {
      if (intervalId) {
        clearInterval(intervalId);
      }
    };
  }, [serviceStatus, wsConnection?.OPEN, serviceId]);

  if (wsConnection) {
    wsConnection.onmessage = (event) => {
      const { status = "", serviceId: wsServiceId = "" } = JSON.parse(
        JSON.parse(event.data)?.message
      );
      if (wsServiceId === serviceId) {
        setServiceStatus(status);
      }
    };
  }
  //   useEffect(()=>{
  // if(serviceStatus ==="active"){
  //   const k = setInterval(() => {
  //     wsConnection?.send(JSON.stringify({"serviceId":serviceId,"type":"userActivePolling"}))
  //   }, 20000);
  // }
  //   },[serviceStatus])

  const handleDeleteService = async () => {
    setIsProcessing(true);
    setError(null);

    try {
      const response = await axiosFetchAuth().delete(`/service/${serviceId}`);
      if (response.status === 200) {
        setServiceStatus("terminated");
      } else {
        throw new Error("Failed to terminate service");
      }
    } catch (err) {
      console.error("Error terminating service:", err);
      setError(
        err instanceof Error ? err.message : "Failed to terminate service"
      );
    } finally {
      setIsProcessing(false);
    }
  };

  const handleReactivateService = async () => {
    setIsProcessing(true);
    setError(null);

    try {
      const response = await axiosFetchAuth().patch(`/service/${serviceId}`);
      if (response.status === 200) {
        setServiceStatus(response.data.service.status);
      } else {
        throw new Error("Failed to reactivate service");
      }
    } catch (err) {
      console.error("Error reactivating service:", err);
      setError(
        err instanceof Error ? err.message : "Failed to reactivate service"
      );
    } finally {
      setIsProcessing(false);
    }
  };

  // Status-based tailwind colors
  const getStatusColors = (status) => {
    switch (status) {
      case "active":
        return "bg-green-100 text-green-800 border-green-300";
      case "terminated":
        return "bg-red-100 text-red-800 border-red-300";
      case "spawning":
        return "bg-yellow-100 text-yellow-800 border-yellow-300";
      default:
        return "bg-gray-100 text-gray-800 border-gray-300";
    }
  };

  if (loading) {
    return (
      <div className="flex flex-col justify-center items-center min-h-screen bg-gradient-to-br from-gray-50 to-gray-100">
        <div className="relative w-20 h-20">
          <div className="absolute top-0 left-0 w-full h-full border-4 border-blue-200 rounded-full animate-ping opacity-75"></div>
          <div className="absolute top-0 left-0 w-full h-full border-4 border-t-blue-500 border-r-transparent border-b-transparent border-l-transparent rounded-full animate-spin"></div>
        </div>
        <p className="mt-6 text-gray-600 font-medium animate-pulse">
          Loading service details...
        </p>
      </div>
    );
  }

  if (error) {
    return (
      <div className="max-w-2xl mx-auto p-8 bg-white shadow-xl rounded-xl mt-10 border-l-4 border-red-500 transform hover:scale-102 transition-all duration-300">
        <div className="bg-red-50 p-6 rounded-lg text-red-700 mb-4">
          <h3 className="font-bold text-xl mb-2">Error</h3>
          <p className="opacity-90">{error}</p>
        </div>
        <button
          onClick={fetchServiceDetails}
          className="px-6 py-3 bg-blue-500 text-white rounded-lg hover:bg-blue-600 transition duration-300 shadow-md hover:shadow-lg transform hover:-translate-y-1"
        >
          Try Again
        </button>
      </div>
    );
  }

  if (!service || !container) {
    return (
      <div className="max-w-2xl mx-auto p-8 bg-white shadow-xl rounded-xl mt-10">
        <p className="text-gray-600 text-center text-lg">
          No service data found.
        </p>
      </div>
    );
  }

  return (
    <div className="max-w-4xl mx-auto p-0 bg-white shadow-2xl rounded-2xl mt-10 overflow-hidden transition-all duration-300">
      {/* Header with status */}
      <div
        className={`p-6 ${getStatusColors(
          serviceStatus
        )} border-b transition-colors duration-500`}
      >
        <div className="flex justify-between items-center">
          <div>
            <h2 className="text-3xl font-bold">{service.name}</h2>
            <p className="text-sm mt-1 opacity-75">
              ID: {service._id.substring(0, 8)}...
            </p>
          </div>
          <div className="flex items-center">
            <div
              className={`w-3 h-3 rounded-full mr-2 ${
                serviceStatus === "active"
                  ? "bg-green-500"
                  : serviceStatus === "spawning"
                  ? "bg-yellow-500 animate-pulse"
                  : "bg-red-500"
              }`}
            ></div>
            <span className="font-medium capitalize">
              {serviceStatus || "Unknown"}
            </span>
          </div>
        </div>
      </div>

      {/* Content split into two columns on larger screens */}
      <div className="p-6 md:grid md:grid-cols-2 md:gap-8">
        {/* Service Section */}
        <div className="mb-8 md:mb-0">
          <h3 className="text-xl font-bold mb-4 text-gray-800 border-b pb-2">
            Service Information
          </h3>

          <div className="space-y-5">
            <div className="bg-gray-50 p-4 rounded-lg hover:shadow-md transition-shadow duration-300">
              <label className="text-sm font-medium text-gray-500">
                Service ID
              </label>
              <p className="text-gray-800 font-mono text-sm mt-1 break-all">
                {service._id}
              </p>
            </div>

            <div className="bg-gray-50 p-4 rounded-lg hover:shadow-md transition-shadow duration-300">
              <label className="text-sm font-medium text-gray-500">
                Linked Container
              </label>
              <p className="text-gray-800 font-mono text-sm mt-1 break-all">
                {service.linkedContainer}
              </p>
            </div>

            <div className="bg-gray-50 p-4 rounded-lg hover:shadow-md transition-shadow duration-300">
              <label className="text-sm font-medium text-gray-500">
                Status Controls
              </label>
              <div className="mt-3">
                {isProcessing ? (
                  <div className="flex items-center">
                    <div className="animate-spin rounded-full h-5 w-5 border-b-2 border-gray-500"></div>
                    <p className="ml-3 text-sm text-gray-600">
                      Processing request...
                    </p>
                  </div>
                ) : serviceStatus === "spawning" ? (
                  <div className="flex items-center px-4 py-2 bg-yellow-100 text-yellow-800 rounded-lg">
                    <svg
                      className="animate-spin -ml-1 mr-3 h-5 w-5 text-yellow-600"
                      xmlns="http://www.w3.org/2000/svg"
                      fill="none"
                      viewBox="0 0 24 24"
                    >
                      <circle
                        className="opacity-25"
                        cx="12"
                        cy="12"
                        r="10"
                        stroke="currentColor"
                        strokeWidth="4"
                      ></circle>
                      <path
                        className="opacity-75"
                        fill="currentColor"
                        d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"
                      ></path>
                    </svg>
                    Spawning in progress
                  </div>
                ) : serviceStatus === "active" ? (
                  <>
                    <button
                      onClick={handleDeleteService}
                      className="w-full px-4 py-2 bg-red-500 text-white rounded-lg hover:bg-red-600 disabled:bg-gray-300 transition-all duration-300 transform hover:scale-105 disabled:transform-none"
                      disabled={isProcessing}
                    >
                      Terminate Service
                    </button>
                    <button
                    onClick={()=>{
                      window.open(`http://${serviceId}-3000.code.iamanshik.online`, "_blank");
                      route.push("http://${serviceId}-8080.code.iamanshik.online")

                    }}
                    >Open Editor</button>
                  </>
                ) : serviceStatus === "terminated" ? (
                  <button
                    onClick={handleReactivateService}
                    className="w-full px-4 py-2 bg-green-500 text-white rounded-lg hover:bg-green-600 disabled:bg-gray-300 transition-all duration-300 transform hover:scale-105 disabled:transform-none"
                    disabled={isProcessing}
                  >
                    Reactivate Service
                  </button>
                ) : (
                  <div className="px-4 py-2 bg-red-100 text-red-800 rounded-lg text-center">
                    Error: Unknown status
                  </div>
                )}
              </div>
            </div>
          </div>
        </div>

        {/* Container Section */}
        <div>
          <h3 className="text-xl font-bold mb-4 text-gray-800 border-b pb-2">
            Container Details
          </h3>

          <div className="space-y-5">
            <div className="flex items-center">
              {container.iconUrl ? (
                container.iconUrl.startsWith("<") ? (
                  <div
                    className="flex-shrink-0 w-16 h-16 mr-4 bg-gray-100 rounded-lg overflow-hidden flex items-center justify-center"
                    dangerouslySetInnerHTML={{ __html: container.iconUrl }}
                  />
                ) : (
                  <div className="flex-shrink-0 mr-4 relative w-16 h-16 bg-gray-100 rounded-lg overflow-hidden shadow-md">
                    <Image
                      src={container.iconUrl}
                      alt={`${container.name} icon`}
                      layout="fill"
                      objectFit="contain"
                      className="p-2"
                      onError={(e) => {
                        e.currentTarget.src = "/api/placeholder/48/48";
                        e.currentTarget.alt = "Icon failed to load";
                      }}
                    />
                  </div>
                )
              ) : (
                <div className="flex-shrink-0 mr-4 w-16 h-16 bg-gray-100 rounded-lg flex items-center justify-center">
                  <svg
                    xmlns="http://www.w3.org/2000/svg"
                    className="h-8 w-8 text-gray-400"
                    fill="none"
                    viewBox="0 0 24 24"
                    stroke="currentColor"
                  >
                    <path
                      strokeLinecap="round"
                      strokeLinejoin="round"
                      strokeWidth={2}
                      d="M5 8h14M5 8a2 2 0 110-4h14a2 2 0 110 4M5 8v10a2 2 0 002 2h10a2 2 0 002-2V8m-9 4h4"
                    />
                  </svg>
                </div>
              )}

              <div>
                <h4 className="font-bold text-lg">{container.name}</h4>
                <div className="inline-block px-2 py-1 bg-blue-100 text-blue-800 text-xs rounded mt-1">
                  {container.stack}
                </div>
              </div>
            </div>

            <div className="bg-gray-50 p-4 rounded-lg hover:shadow-md transition-shadow duration-300">
              <label className="text-sm font-medium text-gray-500">
                Container ID
              </label>
              <p className="text-gray-800 font-mono text-sm mt-1 break-all">
                {container._id}
              </p>
            </div>

            <div className="p-4 rounded-lg border border-dashed border-gray-300 hover:border-blue-300 transition-colors duration-300">
              <div className="flex justify-between items-center">
                <span className="text-sm font-medium text-gray-500">
                  Stack Details
                </span>
                <span className="px-2 py-1 bg-blue-100 text-blue-800 text-xs rounded">
                  {container.stack}
                </span>
              </div>
              <p className="mt-2 text-sm text-gray-600">
                This container is running the <strong>{container.stack}</strong>{" "}
                stack and is linked to service <strong>{service.name}</strong>.
              </p>
            </div>
          </div>
        </div>
      </div>

      {/* Footer */}
      <div className="bg-gray-50 p-4 text-center text-sm text-gray-500 border-t">
        <p>Last updated: {new Date().toLocaleString()}</p>
      </div>
    </div>
  );
}
