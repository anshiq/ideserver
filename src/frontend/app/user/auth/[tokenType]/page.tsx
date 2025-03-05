"use client";
import React, { useEffect, useState } from "react";
import { axiosFetch } from "@/lib/axiosConfig";
import { showNotification } from "@/lib/Notification";
import { useParams, useSearchParams } from "next/navigation";
function page() {
  const {tokenType}  = useParams()
  const token  = useSearchParams().get("token")

 
  if (tokenType === "reset-password") {
    return <ResetPassword token={token} />;
  }
  if (tokenType === "verify-email") {
    return (
      <div>
        <VerifyEmail token={token} />
      </div>
    );
  }
}
const VerifyEmail =  ({ token }) => {
  const k = async () => {
    const data = await axiosFetch().post(
      "/user/verify-user",
      {
        token: token,
      },
      {
        headers: {
          "Content-Type": "application/json",
        },
      }
    );
    setMsg(data.data.data.msg);
  };
  const [msg, setMsg] = useState("");
  useEffect(() => {
    k();
  }, [token]);
  return <>{msg}</>;
};
const ResetPassword = ({ token }) => {
  const [password, setPassword] = useState({
    password: "",
    re_password: "",
  });
  const handleSubmit = (e: React.FormEvent<HTMLFormElement>) => {
    e.preventDefault();
    if (password.password === password.re_password) {
      axiosFetch().post(
        "/user/verify-forgot-token",
        {
          password: password.password,
          token: token,
        },
        {
          headers: {
            "Content-Type": "application/json",
          },
        }
      ).then(data=>{
        if(data.data.success ===true){
           showNotification({ text: data.data.data.msg, color: "green" });
           window.location.href= "/user/auth?type=0"
        }else{
          showNotification({ text: data.data.data.msg, color: "orange" });
        }

      }).catch(e=>{
        console.log(e);
        showNotification({ text: "Internal Server Errror!!", color: "red" });
      })
    } else {
      showNotification({ text: "Password Must be Same..", color: "red" });
    }
  };
  return (
    <>
      <div className="min-h-screen flex items-center justify-center bg-gray-100">
        <div className="max-w-md w-full bg-white rounded-lg shadow-lg overflow-hidden">
          <div className="p-6">
            <form onSubmit={handleSubmit}>
              <div className="mb-4">
                <label
                  htmlFor="name"
                  className="block text-sm font-medium text-gray-700"
                >
                  Enter Your New Password ...{" "}
                </label>
                <input
                  type="password"
                  id="name"
                  name="re_password"
                  placeholder="example@gmail.com"
                  value={password.re_password}
                  onChange={(e) => {
                    setPassword((k) => ({
                      ...k,
                      [e.target.name]: e.target.value,
                    }));
                  }}
                  required
                  minLength={8}
                  className="mt-1 p-2 w-full border rounded-md"
                />
              </div>
              <div className="mb-4">
                <label
                  htmlFor="name"
                  className="block text-sm font-medium text-gray-700"
                >
                  Retype Your New Password ...{" "}
                </label>
                <input
                  type="password"
                  id="name"
                  name="password"
                  placeholder="example@gmail.com"
                  value={password.password}
                  onChange={(e) => {
                    setPassword((k) => ({
                      ...k,
                      [e.target.name]: e.target.value,
                    }));
                  }}
                  required
                  minLength={8}
                  className="mt-1 p-2 w-full border rounded-md"
                />
              </div>
              <button
                type="submit"
                className="w-full p-2 bg-blue-500 text-white rounded-md hover:bg-blue-600"
              >
                Reset Password !!!
              </button>
            </form>
          </div>
        </div>
      </div>
    </>
  );
};

export default page;
