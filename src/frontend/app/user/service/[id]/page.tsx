'use client'

import { axiosFetchAuth } from "@/lib/axiosConfig";
import { use, useEffect, useState } from "react"

export default function GetServiceDetails({ params }: { params: Promise<{ id: string }> }) {
    const { id } = use(params);
    const serviceId = id || ""
    const [serviceData,setServiceData] = useState(null)
    useEffect(()=>{
        axiosFetchAuth().get("/service/"+ serviceId).then(e=>{
            setServiceData(e.data)
        })
    },[id])
    console.log(serviceData)
    return <></>
}