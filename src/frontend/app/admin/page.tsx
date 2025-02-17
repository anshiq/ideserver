export default function Admin() {
    return (
      <div className="min-h-screen bg-gray-100 flex items-center justify-center p-4">
        <form 
          action="/create-pod-schema" 
          method="POST" 
          className="bg-white shadow-md rounded-lg p-6 w-full max-w-md"
        >
          <div className="mb-4">
            <label htmlFor="name" className="block text-gray-700 font-medium mb-2">
              Pod Name
            </label>
            <input 
              type="text" 
              id="name" 
              className="w-full px-3 py-2 border border-gray-300 rounded-md focus:outline-none focus:ring-2 focus:ring-blue-500"
              placeholder="Enter pod name"
            />
          </div>
  
          <div className="mb-4">
            <label htmlFor="stack" className="block text-gray-700 font-medium mb-2">
              Stack
            </label>
            <input 
              type="text" 
              id="stack" 
              className="w-full px-3 py-2 border border-gray-300 rounded-md focus:outline-none focus:ring-2 focus:ring-blue-500"
              placeholder="Enter stack"
            />
          </div>
  
          <div className="mb-6">
            <label htmlFor="schema" className="block text-gray-700 font-medium mb-2">
              Schema
            </label>
            <textarea 
              id="schema" 
              name="yaml" 
              
              className="w-full px-3 py-2 border border-gray-300 rounded-md focus:outline-none focus:ring-2 focus:ring-blue-500"
              placeholder="Enter YAML schema"
            />
          </div>
  
          <button 
            type="submit" 
            className="w-full bg-blue-500 text-white font-medium py-2 px-4 rounded-md hover:bg-blue-600 focus:outline-none focus:ring-2 focus:ring-blue-500"
          >
            Submit
          </button>
        </form>
      </div>
    );
  }