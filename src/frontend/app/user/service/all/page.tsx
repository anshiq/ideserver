"use client";
import React, { useEffect, useState } from "react";
import { axiosFetchAuth } from "@/lib/axiosConfig";

type ServiceType = {
  _id: string;
  name: string;
  linkedContainer: string;
  status: string;
  userId: string;
  __v: number;
};

export default function ServicesList() {
  const [services, setServices] = useState<ServiceType[]>([]);
  const [loading, setLoading] = useState<boolean>(true);

  useEffect(() => {
    const fetchServices = async () => {
      try {
        const response = await axiosFetchAuth().get("/services");
        if (response.data.success) {
          setServices(response.data.services);
        }
      } catch (error) {
        console.error("Error fetching services:", error);
      } finally {
        setLoading(false);
      }
    };

    fetchServices();
  }, []);

  if (loading) return <p className="text-center text-gray-500">Loading services...</p>;

  return (
    <div className="p-6 bg-gray-100 min-h-screen flex items-center justify-center">
      <div className="bg-white p-6 rounded-2xl shadow-xl w-full max-w-3xl">
        <h1 className="text-3xl font-bold text-gray-800 mb-6 text-center">Services</h1>
        <div className="space-y-4">
          {services.map((service) => (
            <div
              key={service._id}
              className="p-4 border rounded-lg shadow-md bg-gray-50 hover:shadow-lg transition"
            >
              <h2 className="text-xl font-semibold text-gray-800">{service.name}</h2>
              <p className="text-gray-600">Status: 
                <span 
                  className={`ml-2 px-2 py-1 rounded text-white ${
                    service.status === "terminated" ? "bg-red-500" : "bg-green-500"
                  }`}
                >
                  {service.status}
                </span>
              </p>
              <p className="text-gray-600 text-sm">Linked Container: {service.linkedContainer}</p>
            </div>
          ))}
        </div>
      </div>
    </div>
  );
}
