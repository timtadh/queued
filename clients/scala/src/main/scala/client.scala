/* queued scala client
 * Author: Tim Henderson
 * Email: tadh@case.edu
 * Copyright 2013 All Right Reserved
 *
 * Redistribution and use in source and binary forms, with or without
 * modification, are permitted provided that the following conditions are met:
 * 
 *  * Redistributions of source code must retain the above copyright notice,
 *    this list of conditions and the following disclaimer.
 *
 *  * Redistributions in binary form must reproduce the above copyright notice,
 *    this list of conditions and the following disclaimer in the documentation
 *    and/or other materials provided with the distribution.
 *
 *  * Neither the name of the queued nor the names of its contributors may be
 *    used to endorse or promote products derived from this software without
 *    specific prior written permission.
 * 
 * THIS SOFTWARE IS PROVIDED BY THE COPYRIGHT HOLDERS AND CONTRIBUTORS "AS IS"
 * AND ANY EXPRESS OR IMPLIED WARRANTIES, INCLUDING, BUT NOT LIMITED TO, THE
 * IMPLIED WARRANTIES OF MERCHANTABILITY AND FITNESS FOR A PARTICULAR PURPOSE
 * ARE DISCLAIMED. IN NO EVENT SHALL THE COPYRIGHT HOLDER OR CONTRIBUTORS BE
 * LIABLE FOR ANY DIRECT, INDIRECT, INCIDENTAL, SPECIAL, EXEMPLARY, OR
 * CONSEQUENTIAL DAMAGES (INCLUDING, BUT NOT LIMITED TO, PROCUREMENT OF
 * SUBSTITUTE GOODS OR SERVICES; LOSS OF USE, DATA, OR PROFITS; OR BUSINESS
 * INTERRUPTION) HOWEVER CAUSED AND ON ANY THEORY OF LIABILITY, WHETHER IN
 * CONTRACT, STRICT LIABILITY, OR TORT (INCLUDING NEGLIGENCE OR OTHERWISE)
 * ARISING IN ANY WAY OUT OF THE USE OF THIS SOFTWARE, EVEN IF ADVISED OF THE
 * POSSIBILITY OF SUCH DAMAGE.
 */

import java.net.Socket;
import java.io;
import java.util.concurrent.LinkedBlockingQueue;
import org.apache.commons.codec.binary.Base64;
import scala.io.Source;

object QueueClient {

  val lines = new LinkedBlockingQueue[String];

  val socket = new Socket("localhost", 8080)
  val writer = new io.OutputStreamWriter(socket.getOutputStream())
  val reader = Source.fromInputStream(socket.getInputStream()).bufferedReader()

  val read_thread = new Thread(new Runnable {
    def run() {
      val source =
      while (!socket.isClosed()) {
        try {
          lines.put(reader.readLine())
        } catch {
          case e: java.net.SocketException => //pass
        }
      }
    }
  })

  def send(msg:String) {
    writer.write(msg)
    writer.flush()
  }

  def get_line():Pair[String, String] = {
    return process_line(lines.take())
  }

  def process_line(line:String):Pair[String, String] = {
    val split = line.split(" ", 2)
    val command = split(0)
    var rest = ""
    if (split.length > 1) {
      rest = new String(Base64.decodeBase64(split(1).getBytes()));
    }
    return new Pair[String, String](command, rest)
  }

  def enque(data:String) {
    send("ENQUE " + new String(Base64.encodeBase64(data.getBytes())) + "\n")
    check_enque_response()
  }

  def check_enque_response() {
    val line = get_line()
    if (line._1 == "ERROR") {
      throw new Exception(line._2)
    } else if (line._1 != "OK") {
      throw new Exception("Bad command recieved")
    }
  }

  def deque():String = {
    send("DEQUE\n")
    return get_deque_response()
  }

  def get_deque_response():String = {
    val line = get_line()
    if (line._1 == "ERROR") {
      if (line._2 == "queue is empty") {
        throw new java.lang.IndexOutOfBoundsException("queue is empty")
      }
      throw new Exception(line._2)
    } else if (line._1 != "ITEM") {
      throw new Exception("Bad command recieved")
    }
    return line._2
  }

  def main(args: Array[String]) {
    read_thread.start()
    try {
      enque("asdf")
      enque("asdf")
      enque("asdf")
      println(deque())
      println(deque())
      println(deque())
      println(deque())
    } catch {
      case e:java.lang.IndexOutOfBoundsException => println("queue empty")
    } finally {
      socket.close()
      read_thread.join()
    }
  }
}

