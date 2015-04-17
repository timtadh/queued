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
package edu.cwru.selab.queued;

import scala.io.Source;

import java.net.Socket;
import java.io;
import java.util.concurrent.LinkedBlockingQueue;
import java.security.MessageDigest;

import org.apache.commons.codec.binary.Base64;


class Queued(host:String, port:Int) {

  val lines = new LinkedBlockingQueue[String];
  val socket = new Socket(host, port)
  val writer = new io.OutputStreamWriter(socket.getOutputStream())
  val reader = Source.fromInputStream(socket.getInputStream()).bufferedReader()

  val read_thread = new Thread(new Runnable {
    def run() {
      val source =
      while (!socket.isClosed()) {
        try {
          val line = reader.readLine()
          if (line != null) {
              lines.put(line)
          } else {
              lines.put("ERROR queued died")
              socket.close()
              return
          }
        } catch {
          case e: java.net.SocketException => //pass
        }
      }
    }
  })

  read_thread.start()

  def stop() {
    socket.close()
    read_thread.join()
  }

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
      rest = split(1)
    }
    return new Pair[String, String](command, rest)
  }

  def use(name:String) {
    send("USE " + name + "\n")
    check_use_response()
  }

  def check_use_response() {
    val line = get_line()
    if (line._1 == "ERROR") {
      val err = new String(Base64.decodeBase64(line._2.getBytes()));
      throw new Exception(err)
    } else if (line._1 != "OK") {
      throw new Exception("Bad command recieved")
    }
  }

  def enque(data:String) {
    send("ENQUE " + new String(Base64.encodeBase64(data.getBytes())) + "\n")
    check_enque_response()
  }

  def check_enque_response() {
    val line = get_line()
    if (line._1 == "ERROR") {
      val err = new String(Base64.decodeBase64(line._2.getBytes()));
      throw new Exception(err)
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
      val err = new String(Base64.decodeBase64(line._2.getBytes()));
      if (err == "queue is empty") {
        throw new java.lang.IndexOutOfBoundsException("queue is empty")
      }
      throw new Exception(err)
    } else if (line._1 != "ITEM") {
      throw new Exception("Bad command recieved")
    }
    val data = new String(Base64.decodeBase64(line._2.getBytes()));
    return data
  }

  def hash(data:String):String = {
    val h = MessageDigest.getInstance("SHA-256")
    val d = h.digest(data.getBytes())
    return new String(Base64.encodeBase64(d))
  }

  def has(data:String):Boolean = {
    send("HAS " + hash(data) + "\n")
    get_has_response()
  }

  def get_has_response():Boolean = {
    val line = get_line()
    if (line._1 == "ERROR") {
      val err = new String(Base64.decodeBase64(line._2.getBytes()));
      throw new Exception(err)
    } else if (line._1 == "TRUE") {
      return true
    } else if (line._1 == "FALSE") {
      return false
    }
    throw new Exception("Bad command recieved")
  }

  def size():Int = {
    send("SIZE" + "\n")
    get_size_response()
  }

  def get_size_response():Int = {
    val line = get_line()
    if (line._1 == "ERROR") {
      val err = new String(Base64.decodeBase64(line._2.getBytes()));
      throw new Exception(err)
    } else if (line._1 != "SIZE") {
      throw new Exception("Bad command recieved")
    }
    line._2.toInt
  }
}

object MainQueued {
  def main(args: Array[String]) {
    val queue = new Queued("localhost", 9001)
    try {
      println(queue.size())
      queue.enque("asdf")
      queue.enque("asf")
      queue.enque("assdf")
      queue.use("wizard")
      println(queue.size())
      queue.use("default")
      println(queue.size())
      println(queue.has("asdf"))
      println(queue.has("asf"))
      println(queue.has("assdf"))
      println(queue.deque())
      println(queue.deque())
      println(queue.size())
      println(queue.deque())
      println(queue.size())
      println(queue.has("asdf"))
      println(queue.has("asf"))
      println(queue.has("assdf"))
      println(queue.deque())
    } catch {
      case e:java.lang.IndexOutOfBoundsException => println("queue empty")
    } finally {
      queue.stop()
    }
  }
}
